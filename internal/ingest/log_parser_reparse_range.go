package ingest

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qianlima-666/nginxpulse/internal/config"
	"github.com/qianlima-666/nginxpulse/internal/ingest/source"
	"github.com/sirupsen/logrus"
)

func (p *LogParser) TriggerRangeReparse(websiteID string, start, end time.Time) error {
	if strings.TrimSpace(websiteID) == "" {
		return fmt.Errorf("按时间段重解析仅支持单个站点")
	}
	if !end.After(start) {
		return fmt.Errorf("时间范围无效")
	}

	if p.demoMode {
		if err := p.repo.ClearLogsForWebsiteRange(websiteID, start, end); err != nil {
			return err
		}
		return p.repo.RebuildWebsiteDerivedData(websiteID)
	}

	if !startIPParsingWithStage(parseStageReparse) {
		return ErrParsingInProgress
	}

	if err := p.repo.ClearLogsForWebsiteRange(websiteID, start, end); err != nil {
		finishIPParsing()
		return err
	}

	go func() {
		defer finishIPParsing()

		setParsingTotalBytes(p.calculateTotalBytesToReparseRange([]string{websiteID}))
		setParsingWebsiteID(websiteID)
		defer setParsingWebsiteID("")

		if err := p.scanWebsiteRange(websiteID, parseWindow{
			minTs: start.Unix(),
			maxTs: end.Unix(),
		}); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"website": websiteID,
				"start":   start.Format(time.RFC3339),
				"end":     end.Format(time.RFC3339),
			}).Error("按时间段重解析日志失败")
			p.notifyLogParsing(websiteID, "", "按时间段重解析", err)
		}

		if err := p.repo.RebuildWebsiteDerivedData(websiteID); err != nil {
			logrus.WithError(err).WithField("website", websiteID).Error("按时间段重解析后重建衍生数据失败")
			p.notifyDatabaseWrite(websiteID, "重建衍生统计", err)
		}

		p.refreshWebsiteRanges(websiteID)
		p.updateState()
	}()

	return nil
}

func (p *LogParser) scanWebsiteRange(websiteID string, window parseWindow) error {
	website, ok := config.GetWebsiteByID(websiteID)
	if !ok {
		return fmt.Errorf("站点不存在: %s", websiteID)
	}

	parserResult := EmptyParserResult(website.Name, websiteID)
	startTime := time.Now()

	if len(website.Sources) > 0 {
		if err := p.scanSourcesForWindow(websiteID, website, &parserResult, window); err != nil {
			return err
		}
	} else {
		if _, err := p.getLineParser(websiteID); err != nil {
			return err
		}

		logPath := website.LogPath
		if strings.Contains(logPath, "*") {
			matches, err := filepath.Glob(logPath)
			if err != nil {
				return fmt.Errorf("解析日志路径模式失败: %w", err)
			}
			if len(matches) == 0 {
				return fmt.Errorf("日志路径模式 %s 未匹配到任何文件", logPath)
			}
			for _, matchPath := range matches {
				if err := p.scanSingleFileForWindow(websiteID, matchPath, "", &parserResult, window); err != nil {
					return err
				}
			}
		} else {
			if err := p.scanSingleFileForWindow(websiteID, logPath, "", &parserResult, window); err != nil {
				return err
			}
		}
	}

	parserResult.Duration = time.Since(startTime)
	if parserResult.Error != nil {
		return parserResult.Error
	}
	return nil
}

func (p *LogParser) scanSingleFileForWindow(
	websiteID string,
	logPath string,
	sourceID string,
	parserResult *ParserResult,
	window parseWindow,
) error {
	file, err := os.Open(logPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var (
		reader io.Reader = file
		closer interface{ Close() error }
	)
	if isGzipFile(logPath) {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return err
		}
		reader = gzReader
		closer = gzReader
	}
	if closer != nil {
		defer closer.Close()
	}

	sourceCtx := fileParseSourceContext(logPath, 0)
	sourceCtx.sourceID = sourceID
	entriesCount, _, _, _ := p.parseLogLines(reader, websiteID, sourceCtx, parserResult, window)
	if entriesCount > 0 {
		logrus.Infof("网站 %s 的日志文件 %s 按时间段重解析完成，解析了 %d 条记录", websiteID, logPath, entriesCount)
	}
	return nil
}

func (p *LogParser) scanSourcesForWindow(
	websiteID string,
	website config.WebsiteConfig,
	parserResult *ParserResult,
	window parseWindow,
) error {
	ctx := context.Background()
	for _, srcCfg := range website.Sources {
		if _, err := p.getLineParserForSource(websiteID, srcCfg.ID); err != nil {
			return err
		}
		src, err := source.NewFromConfig(websiteID, srcCfg)
		if err != nil {
			return err
		}

		mode := strings.ToLower(strings.TrimSpace(srcCfg.Mode))
		if mode == "" {
			mode = "poll"
		}
		if mode == "stream" {
			continue
		}

		targets, err := src.ListTargets(ctx)
		if err != nil {
			return err
		}
		for _, target := range targets {
			if err := p.scanTargetForWindow(ctx, websiteID, src, target, parserResult, window); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *LogParser) scanTargetForWindow(
	ctx context.Context,
	websiteID string,
	src source.LogSource,
	target source.TargetRef,
	parserResult *ParserResult,
	window parseWindow,
) error {
	reader, err := src.OpenRange(ctx, target, 0, -1)
	if err != nil {
		return err
	}
	if reader == nil {
		return nil
	}
	defer reader.Close()

	var entriesCount int
	if target.Meta.Compressed {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		sourceCtx := targetParseSourceContext(target.SourceID, target.Key, 0)
		entriesCount, _, _, _ = p.parseLogLines(gzReader, websiteID, sourceCtx, parserResult, window)
		gzReader.Close()
	} else {
		sourceCtx := targetParseSourceContext(target.SourceID, target.Key, 0)
		entriesCount, _, _, _ = p.parseLogLines(reader, websiteID, sourceCtx, parserResult, window)
	}

	if entriesCount > 0 {
		logrus.Infof("网站 %s 的远端目标 %s 按时间段重解析完成，解析了 %d 条记录", websiteID, target.Key, entriesCount)
	}
	return nil
}

func (p *LogParser) calculateTotalBytesToReparseRange(websiteIDs []string) int64 {
	var total int64

	for _, id := range websiteIDs {
		website, ok := config.GetWebsiteByID(id)
		if !ok {
			continue
		}
		if len(website.Sources) > 0 {
			continue
		}

		logPath := website.LogPath
		if strings.Contains(logPath, "*") {
			matches, err := filepath.Glob(logPath)
			if err != nil {
				logrus.Warnf("解析日志路径模式 %s 失败: %v", logPath, err)
				continue
			}
			for _, matchPath := range matches {
				total += p.fullScanableBytes(matchPath)
			}
			continue
		}

		total += p.fullScanableBytes(logPath)
	}

	return total
}

func (p *LogParser) fullScanableBytes(logPath string) int64 {
	fileInfo, err := os.Stat(logPath)
	if err != nil {
		return 0
	}
	return fileInfo.Size()
}
