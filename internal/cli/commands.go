package cli

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/qianlima-666/nginxpulse/internal/config"
	"github.com/qianlima-666/nginxpulse/internal/version"
)

// HandleAppConfig 处理应用程序配置初始化和命令行参数
func ProcessCliCommands() bool {
	// 命令行参数
	cleanApp := flag.Bool("clean", false, "清理nginxpulse服务、释放端口和删除数据")
	showVer := flag.Bool("v", false, "显示版本信息")
	flag.Parse()

	// 显示版本信息
	if *showVer {
		showVersion()
		return true
	}

	// 清理服务
	if *cleanApp {
		cleanService()
		return true
	}

	// 检查配置文件
	if exit := initConfig(); exit {
		return true
	}

	// 验证配置文件是否完整有效
	if exit := validateConfig(); exit {
		return true
	}

	// 初始化目录
	if exit := initDirs(); exit {
		return true
	}

	// 不需要退出，继续运行
	return false
}

// showVersion 显示版本信息
func showVersion() {
	fmt.Printf("构建时间: %s\n", version.BuildTime)
	fmt.Printf("Git 提交: %s\n", version.GitCommit)
}

func initConfig() bool {
	if _, err := os.Stat(config.ConfigFile); err == nil {
		return false
	}

	if config.HasEnvConfigSource() {
		return false
	}

	fmt.Fprintf(os.Stderr, "未找到配置文件: %s\n", config.ConfigFile)
	fmt.Fprintln(os.Stderr, "将进入初始化配置模式，可在页面完成配置")
	return false
}

// initDirs 初始化目录
func initDirs() bool {
	dirs := []string{
		config.DataDir,
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "初始化目录失败: %v\n", err)
				return true
			}
		}
	}
	return false
}

// validateConfig 验证配置文件是否完整有效
func validateConfig() bool {

	// 读取配置
	cfg, err := config.ReadRawConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取配置文件失败: %v\n", err)
		fmt.Fprintf(os.Stderr, "请修正配置问题后重新启动服务\n")
		return true
	}

	if config.NeedsSetup() {
		return false
	}
	result := config.ValidateConfig(cfg, config.ValidateOptions{
		CheckPaths: !cfg.System.DemoMode,
	})
	if len(result.Errors) == 0 {
		return false
	}
	fmt.Fprintln(os.Stderr, "配置文件错误:")
	for _, item := range result.Errors {
		if item.Field == "" {
			fmt.Fprintf(os.Stderr, " - %s\n", item.Message)
		} else {
			fmt.Fprintf(os.Stderr, " - %s: %s\n", item.Field, item.Message)
		}
	}
	fmt.Fprintln(os.Stderr, "请修正配置问题后重新启动服务")
	return true
}

// cleanService 清理 nginxpulse 服务、释放端口和删除数据
func cleanService() {
	fmt.Println("开始清理nginxpulse服务...")

	findAndTerminateProcesses("nginxpulse")

	// 清理数据目录
	fmt.Println("开始清理数据目录...")
	if err := os.RemoveAll(config.DataDir); err != nil {
		fmt.Printf("清理数据目录失败: %v\n", err)
	}
	fmt.Println("清理工作完成")
}

// findAndTerminateProcesses 查找并终止指定进程
func findAndTerminateProcesses(processName string) {
	// 获取当前进程和父进程的PID
	skipPID := os.Getpid()
	ppid := os.Getppid()

	// 查找并终止进程
	cmd := exec.Command("pgrep", "-f", processName)
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		fmt.Printf("没有发现 %s 进程\n", processName)
		return
	}

	for _, pidStr := range strings.Split(
		strings.TrimSpace(string(output)), "\n") {
		// 解析PID
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err != nil || pid == skipPID || pid == ppid {
			continue
		}

		// 终止进程
		if proc, err := os.FindProcess(pid); err == nil {
			fmt.Printf("正在终止进程 (PID: %d)\n", pid)
			proc.Signal(syscall.SIGKILL)
		}
	}
}
