package version

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/gosuri/uitable"
)

var (
	// gitVersion 是语义化的版本号.
	gitVersion = ""
	// buildDate 是 ISO8601 格式的构建时间, $(date -u +'%Y-%m-%dT%H:%M:%SZ') 命令的输出.
	buildDate = ""
	// gitCommit 是 Git 的 SHA1 值，$(git rev-parse HEAD) 命令的输出.
	gitCommit = ""
	// gitTreeState 代表构建时 Git 仓库的状态，可能的值有：clean, dirty.
	gitTreeState = ""
)

// Info 包含了版本信息.
type Info struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

// String 返回人性化的版本信息字符串.
func (info Info) String() string {
	return info.GitVersion
}

// ToJSON 以 JSON 格式返回版本信息.
func (info Info) ToJSON() string {
	s, _ := json.Marshal(info)

	return string(s)
}

// Text 将版本信息编码为 UTF-8 格式的文本，并返回.
func (info Info) Text() string {
	table := uitable.New()
	table.RightAlign(0)
	table.MaxColWidth = 80
	table.Separator = " "
	table.AddRow("gitVersion:", info.GitVersion)
	table.AddRow("gitCommit:", info.GitCommit)
	table.AddRow("gitTreeState:", info.GitTreeState)
	table.AddRow("buildDate:", info.BuildDate)
	table.AddRow("goVersion:", info.GoVersion)
	table.AddRow("compiler:", info.Compiler)
	table.AddRow("platform:", info.Platform)

	return table.String()
}

// Get 返回详尽的代码库版本信息，用来标明二进制文件由哪个版本的代码构建.
func Get() Info {
	// 以下变量通常由 -ldflags 进行设置
	return Info{
		GitVersion:   gitVersion,
		GitCommit:    gitCommit,
		GitTreeState: gitTreeState,
		BuildDate:    buildDate,
		GoVersion:    runtime.Version(),
		Compiler:     runtime.Compiler,
		Platform:     fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func GetFromDebugInfo(modulePath string) Info {
	info := Info{
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}

	// 尝试从构建信息中获取版本
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		mod := findModule(buildInfo, modulePath)
		if mod == nil {
			info.GitVersion = "unknown"
		}
		if mod.Replace != nil {
			mod = mod.Replace
		}
		info.GitVersion = mod.Version

		// 从构建设置中获取更多信息
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				info.GitCommit = setting.Value
			case "vcs.time":
				info.BuildDate = setting.Value
			case "vcs.modified":
				info.GitTreeState = "clean"
				if setting.Value == "true" {
					info.GitTreeState = "dirty"
				}
			}
		}
	}

	return info
}

func findModule(info *debug.BuildInfo, modulePath string) *debug.Module {
	if info.Main.Path == modulePath {
		return &info.Main
	}
	for _, dep := range info.Deps {
		if dep.Path == modulePath {
			return dep
		}
	}
	return nil
}
