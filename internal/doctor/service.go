package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/junghoonkye/tossinvest-cli/internal/auth"
	"github.com/junghoonkye/tossinvest-cli/internal/config"
	"github.com/junghoonkye/tossinvest-cli/internal/permissions"
	"github.com/junghoonkye/tossinvest-cli/internal/version"
)

type CheckStatus string

const (
	CheckOK   CheckStatus = "ok"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
	CheckInfo CheckStatus = "info"
)

type Check struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Summary string      `json:"summary"`
	Detail  string      `json:"detail,omitempty"`
}

type AuthReport struct {
	PythonBinary string      `json:"python_binary"`
	HelperDir    string      `json:"helper_dir"`
	Session      auth.Status `json:"session"`
	Checks       []Check     `json:"checks"`
}

type Report struct {
	Version    version.Info       `json:"version"`
	GoVersion  string             `json:"go_version"`
	OS         string             `json:"os"`
	Arch       string             `json:"arch"`
	Paths      config.Paths       `json:"paths"`
	Config     config.Status      `json:"config"`
	Permission permissions.Status `json:"permission"`
	Auth       AuthReport         `json:"auth"`
	Checks     []Check            `json:"checks"`
}

type authStatusReader interface {
	Status(context.Context) (auth.Status, error)
}

type permissionStatusReader interface {
	Status(context.Context) (permissions.Status, error)
}

type Service struct {
	paths       config.Paths
	configState config.Status
	loginConfig auth.LoginConfig
	authService authStatusReader
	permService permissionStatusReader
}

func NewService(paths config.Paths, configState config.Status, loginConfig auth.LoginConfig, authService authStatusReader, permService permissionStatusReader) *Service {
	return &Service{
		paths:       paths,
		configState: configState,
		loginConfig: loginConfig,
		authService: authService,
		permService: permService,
	}
}

func (s *Service) Run(ctx context.Context) (Report, error) {
	authReport, err := s.RunAuth(ctx)
	if err != nil {
		return Report{}, err
	}

	permissionStatus, err := s.permService.Status(ctx)
	if err != nil {
		return Report{}, err
	}

	checks := []Check{
		checkPath("config_dir", s.paths.ConfigDir),
		checkPath("cache_dir", s.paths.CacheDir),
		checkFile("config_file", s.paths.ConfigFile),
		checkFile("session_file", s.paths.SessionFile),
		checkFile("permission_file", s.paths.PermissionFile),
		checkFile("lineage_file", s.paths.LineageFile),
		checkTradingConfig(s.configState),
		checkLiveOrderActions(s.configState),
		checkDangerousAutomation(s.configState),
		checkLegacyConfig(s.configState),
		checkPermission(permissionStatus),
		{
			Name:    "trading_scope",
			Status:  CheckInfo,
			Summary: "trading support is intentionally narrow and still beta",
			Detail:  "Currently validated for US/KR buy/sell limit + US fractional (market) orders in KRW, plus same-day pending cancel. Sell requires `trading.sell=true`, KR requires `trading.kr=true`, fractional requires `trading.fractional=true`. Amend still needs more live verification.",
		},
	}

	return Report{
		Version:    version.Current(),
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Paths:      s.paths,
		Config:     s.configState,
		Permission: permissionStatus,
		Auth:       authReport,
		Checks:     checks,
	}, nil
}

func (s *Service) RunAuth(ctx context.Context) (AuthReport, error) {
	sessionStatus, err := s.authService.Status(ctx)
	if err != nil {
		return AuthReport{}, err
	}

	checks := []Check{
		checkPythonBinary(s.loginConfig.PythonBin),
		checkPath("auth_helper_dir", s.loginConfig.HelperDir),
		checkPythonModule(s.loginConfig, "tossctl_auth_helper", "auth helper module is importable", "auth helper module is not importable"),
		checkPythonModule(s.loginConfig, "playwright", "python playwright package is installed", "python playwright package is not installed"),
		checkChromium(s.loginConfig),
		checkSession(sessionStatus),
	}

	return AuthReport{
		PythonBinary: s.loginConfig.PythonBin,
		HelperDir:    s.loginConfig.HelperDir,
		Session:      sessionStatus,
		Checks:       checks,
	}, nil
}

func checkPath(name, path string) Check {
	info, err := os.Stat(path)
	switch {
	case err == nil && info.IsDir():
		return Check{Name: name, Status: CheckOK, Summary: fmt.Sprintf("%s exists", path)}
	case err == nil:
		return Check{Name: name, Status: CheckWarn, Summary: fmt.Sprintf("%s exists but is not a directory", path)}
	case errors.Is(err, os.ErrNotExist):
		return Check{Name: name, Status: CheckWarn, Summary: fmt.Sprintf("%s does not exist yet", path)}
	default:
		return Check{Name: name, Status: CheckWarn, Summary: fmt.Sprintf("could not inspect %s", path), Detail: err.Error()}
	}
}

func checkFile(name, path string) Check {
	info, err := os.Stat(path)
	switch {
	case err == nil && !info.IsDir():
		return Check{Name: name, Status: CheckOK, Summary: fmt.Sprintf("%s exists", path)}
	case err == nil:
		return Check{Name: name, Status: CheckWarn, Summary: fmt.Sprintf("%s exists but is a directory", path)}
	case errors.Is(err, os.ErrNotExist):
		return Check{Name: name, Status: CheckInfo, Summary: fmt.Sprintf("%s does not exist yet", path)}
	default:
		return Check{Name: name, Status: CheckWarn, Summary: fmt.Sprintf("could not inspect %s", path), Detail: err.Error()}
	}
}

func checkPythonBinary(pythonBin string) Check {
	path, err := exec.LookPath(pythonBin)
	if err != nil {
		return Check{
			Name:    "python_binary",
			Status:  CheckFail,
			Summary: fmt.Sprintf("%s was not found in PATH", pythonBin),
			Detail:  "Install Python 3.11+ or set TOSSCTL_AUTH_HELPER_PYTHON.",
		}
	}

	return Check{
		Name:    "python_binary",
		Status:  CheckOK,
		Summary: fmt.Sprintf("using %s", path),
	}
}

func checkPythonModule(cfg auth.LoginConfig, module, successSummary, failSummary string) Check {
	path, err := exec.LookPath(cfg.PythonBin)
	if err != nil {
		return Check{
			Name:    module,
			Status:  CheckFail,
			Summary: failSummary,
			Detail:  "Python is not available.",
		}
	}

	cmd := exec.Command(path, "-c", "import "+module)
	cmd.Dir = cfg.HelperDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Check{
			Name:    module,
			Status:  CheckWarn,
			Summary: failSummary,
			Detail:  string(output),
		}
	}

	return Check{
		Name:    module,
		Status:  CheckOK,
		Summary: successSummary,
	}
}

func checkChromium(cfg auth.LoginConfig) Check {
	path, err := exec.LookPath(cfg.PythonBin)
	if err != nil {
		return Check{
			Name:    "chromium",
			Status:  CheckFail,
			Summary: "chromium check skipped because python is unavailable",
		}
	}

	script := `import os
from playwright.sync_api import sync_playwright
p = sync_playwright().start()
chromium_path = p.chromium.executable_path
p.stop()
if chromium_path and os.path.exists(chromium_path):
    print(chromium_path)
`
	cmd := exec.Command(path, "-c", script)
	cmd.Dir = cfg.HelperDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Check{
			Name:    "chromium",
			Status:  CheckWarn,
			Summary: "playwright chromium is not ready",
			Detail:  chromiumFailureDetail(string(output)),
		}
	}
	chromiumPath := strings.TrimSpace(string(output))
	if chromiumPath == "" {
		return Check{
			Name:    "chromium",
			Status:  CheckWarn,
			Summary: "playwright chromium is not ready",
			Detail:  fmt.Sprintf("Run `%s -m playwright install chromium`.", cfg.PythonBin),
		}
	}

	return Check{
		Name:    "chromium",
		Status:  CheckOK,
		Summary: "playwright chromium is installed",
		Detail:  chromiumPath,
	}
}

func chromiumFailureDetail(output string) string {
	detail := strings.TrimSpace(output)
	if detail == "" {
		return "Run `python3 -m playwright install chromium`."
	}
	if strings.Contains(detail, "Executable doesn't exist") {
		return "Run `python3 -m playwright install chromium`."
	}
	return firstLine(detail)
}

func firstLine(value string) string {
	if idx := strings.IndexByte(value, '\n'); idx >= 0 {
		return value[:idx]
	}
	return value
}

func checkSession(status auth.Status) Check {
	switch {
	case !status.Active:
		return Check{
			Name:    "session",
			Status:  CheckInfo,
			Summary: "no stored session",
			Detail:  "Run `tossctl auth login` after your local auth environment is ready.",
		}
	case status.Validated && status.Valid:
		return Check{
			Name:    "session",
			Status:  CheckOK,
			Summary: "stored session is valid",
		}
	case status.Validated && !status.Valid:
		return Check{
			Name:    "session",
			Status:  CheckWarn,
			Summary: "stored session is no longer valid",
			Detail:  status.ValidationError,
		}
	case status.Expired:
		return Check{
			Name:    "session",
			Status:  CheckWarn,
			Summary: "stored session looks expired",
		}
	default:
		return Check{
			Name:    "session",
			Status:  CheckOK,
			Summary: "stored session exists",
		}
	}
}

func checkPermission(status permissions.Status) Check {
	switch {
	case status.Active:
		return Check{Name: "trading_permission", Status: CheckOK, Summary: "temporary trading permission is active"}
	case status.Expired:
		return Check{Name: "trading_permission", Status: CheckInfo, Summary: "temporary trading permission has expired"}
	default:
		return Check{Name: "trading_permission", Status: CheckInfo, Summary: "no active trading permission grant"}
	}
}

func checkTradingConfig(status config.Status) Check {
	if !status.Exists {
		return Check{
			Name:    "trading_config",
			Status:  CheckInfo,
			Summary: "config file does not exist yet; trading actions default to disabled",
			Detail:  "Run `tossctl config init` to create config.json and enable only the actions you want.",
		}
	}

	enabled := status.Trading.EnabledActions()
	if len(enabled) == 0 {
		return Check{
			Name:    "trading_config",
			Status:  CheckInfo,
			Summary: "config file exists, but all trading actions are disabled",
			Detail:  "Edit config.json to explicitly allow the actions you want to use.",
		}
	}

	return Check{
		Name:    "trading_config",
		Status:  CheckOK,
		Summary: "one or more trading actions are enabled in config",
		Detail:  strings.Join(enabled, ", "),
	}
}

func checkLiveOrderActions(status config.Status) Check {
	if !status.Exists || !status.Trading.AllowLiveOrderActions {
		return Check{
			Name:    "live_order_actions",
			Status:  CheckInfo,
			Summary: "real account-changing order actions are blocked",
			Detail:  "Set `trading.allow_live_order_actions=true` only if you intend to let `place`, `cancel`, or `amend` reach the broker.",
		}
	}

	return Check{
		Name:    "live_order_actions",
		Status:  CheckWarn,
		Summary: "real account-changing order actions are enabled",
		Detail:  "Live `place`, `cancel`, and `amend` can execute after the remaining permission and confirmation gates pass.",
	}
}

func checkDangerousAutomation(status config.Status) Check {
	enabled := status.Trading.DangerousAutomation.EnabledActions()
	if len(enabled) == 0 {
		return Check{
			Name:    "dangerous_automation",
			Status:  CheckInfo,
			Summary: "no risky broker branches will be auto-continued",
		}
	}

	return Check{
		Name:    "dangerous_automation",
		Status:  CheckWarn,
		Summary: "risky broker branch automation is enabled",
		Detail:  strings.Join(enabled, ", ") + " (only has effect when matching branch handlers exist in the current build)",
	}
}

func checkLegacyConfig(status config.Status) Check {
	if !status.Exists {
		return Check{
			Name:    "legacy_config",
			Status:  CheckInfo,
			Summary: "no config file is present, so no legacy translation is needed",
		}
	}

	if len(status.LegacyFields) == 0 {
		return Check{
			Name:    "legacy_config",
			Status:  CheckInfo,
			Summary: "config is already using the current trading policy keys",
		}
	}

	return Check{
		Name:    "legacy_config",
		Status:  CheckWarn,
		Summary: "legacy trading config keys were translated into the current policy model",
		Detail:  strings.Join(status.LegacyFields, ", ") + " -> trading.allow_live_order_actions",
	}
}
