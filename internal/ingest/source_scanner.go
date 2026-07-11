package ingest

import (
	"compress/gzip"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/qianlima-666/nginxpulse/internal/config"
	"github.com/qianlima-666/nginxpulse/internal/ingest/source"
	"github.com/sirupsen/logrus"
)

func (p *LogParser) scanSources(websiteID string, website config.WebsiteConfig, parserResult *ParserResult) {
	ctx := context.Background()
	for _, srcCfg := range website.Sources {
		if _, err := p.getLineParserForSource(websiteID, srcCfg.ID); err != nil {
			parserResult.Success = false
			parserResult.Error = err
			continue
		}
		src, err := source.NewFromConfig(websiteID, srcCfg)
		if err != nil {
			parserResult.Success = false
			parserResult.Error = err
			continue
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
			parserResult.Success = false
			parserResult.Error = err
			continue
		}
		for _, target := range targets {
			if err := p.scanTarget(ctx, websiteID, src, target, parserResult); err != nil {
				parserResult.Success = false
				parserResult.Error = err
			}
		}
	}
}

func (p *LogParser) scanTarget(
	ctx context.Context,
	websiteID string,
	src source.LogSource,
	target source.TargetRef,
	parserResult *ParserResult,
) error {
	targetKey := buildTargetStateKey(target.SourceID, target.Key)
	state, ok := p.getTargetState(websiteID, targetKey)

	meta := target.Meta
	if meta.Size == 0 && meta.ModTime.IsZero() && meta.ETag == "" {
		updated, err := src.Stat(ctx, target)
		if err != nil {
			return err
		}
		meta = updated
	}

	if !ok || state.RecentCutoffTs == 0 {
		state.RecentCutoffTs = time.Now().AddDate(0, 0, -recentLogWindowDays).Unix()
	}

	reset := false
	if ok {
		if meta.Size > 0 && state.LastSize > 0 && meta.Size < state.LastSize {
			reset = true
		}
		if meta.ETag != "" && state.LastETag != "" && meta.ETag != state.LastETag && meta.Size <= state.LastSize {
			reset = true
		}
		if state.LastOffset > 0 && meta.Size > 0 && state.LastOffset > meta.Size {
			reset = true
		}
	}
	if reset {
		state = TargetState{RecentCutoffTs: state.RecentCutoffTs}
		ok = false
	}

	needsFullScan := meta.Compressed
	if needsFullScan && ok {
		sameETag := meta.ETag != "" && meta.ETag == state.LastETag
		sameMod := meta.ETag == "" && meta.Size == state.LastSize && meta.ModTime.Unix() == state.LastModTime
		if meta.Size == state.LastSize && (sameETag || sameMod) {
			return nil
		}
	}

	startOffset := state.LastOffset
	if !ok || needsFullScan {
		startOffset = 0
	}

	if !needsFullScan && meta.Size > 0 && startOffset >= meta.Size {
		state.LastSize = meta.Size
		state.LastETag = meta.ETag
		state.LastModTime = meta.ModTime.Unix()
		p.setTargetState(websiteID, targetKey, state)
		return nil
	}

	reader, err := src.OpenRange(ctx, target, startOffset, -1)
	if err != nil {
		if errors.Is(err, source.ErrRangeNotSupported) && startOffset > 0 {
			reader, err = src.OpenRange(ctx, target, 0, -1)
			if err != nil {
				return err
			}
			if skipErr := skipReaderBytes(reader, startOffset); skipErr != nil {
				reader.Close()
				return skipErr
			}
		} else {
			return err
		}
	}
	if reader == nil {
		return nil
	}
	defer reader.Close()

	window := parseWindow{}
	if !ok {
		window = parseWindow{minTs: state.RecentCutoffTs}
	}

	var (
		entriesCount int
		bytesRead    int64
		minTs        int64
		maxTs        int64
	)

	if needsFullScan {
		gzReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		sourceCtx := targetParseSourceContext(target.SourceID, target.Key, 0)
		entriesCount, bytesRead, minTs, maxTs = p.parseLogLines(gzReader, websiteID, sourceCtx, parserResult, window)
		gzReader.Close()
	} else {
		sourceCtx := targetParseSourceContext(target.SourceID, target.Key, startOffset)
		entriesCount, bytesRead, minTs, maxTs = p.parseLogLines(reader, websiteID, sourceCtx, parserResult, window)
	}

	updateTargetParsedRange(&state, minTs, maxTs)
	if meta.ModTime.Unix() > state.LastTimestamp {
		state.LastTimestamp = meta.ModTime.Unix()
	}

	if needsFullScan {
		state.LastOffset = meta.Size
		state.BackfillDone = true
	} else {
		state.LastOffset = startOffset + bytesRead
		state.BackfillDone = true
	}
	state.LastSize = meta.Size
	state.LastETag = meta.ETag
	state.LastModTime = meta.ModTime.Unix()
	p.setTargetState(websiteID, targetKey, state)

	if entriesCount > 0 {
		logrus.Infof("网站 %s 的远端目标 %s 扫描完成，解析了 %d 条记录", websiteID, target.Key, entriesCount)
	}
	return nil
}

func buildTargetStateKey(sourceID, key string) string {
	if sourceID == "" {
		return key
	}
	return sourceID + ":" + key
}

func updateTargetParsedRange(state *TargetState, minTs, maxTs int64) {
	if minTs > 0 && (state.ParsedMinTs == 0 || minTs < state.ParsedMinTs) {
		state.ParsedMinTs = minTs
	}
	if maxTs > 0 && maxTs > state.ParsedMaxTs {
		state.ParsedMaxTs = maxTs
	}
	if state.FirstTimestamp == 0 || (minTs > 0 && minTs < state.FirstTimestamp) {
		state.FirstTimestamp = minTs
	}
	if maxTs > 0 && maxTs > state.LastTimestamp {
		state.LastTimestamp = maxTs
	}
}
