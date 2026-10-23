package log

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/coderyw/gomicro-util/library/env"
	srv_config "github.com/coderyw/gomicro-util/model/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// defaultLogger 默认的zap日志记录器实例
	defaultLogger *zap.Logger
)

// 包初始化函数，使用默认配置初始化日志记录器
func init() {
	InitDefaultLogger(srv_config.LogCfg{Level: -1}, "", "")
}

// censor 敏感信息过滤函数，用于屏蔽日志中的敏感数据
// 当遇到密码字段时，返回星号代替实际值
var censor = func(key, value string) string {
	if key == "password" {
		return "****"
	}
	return value
}

// InitDefaultLogger 初始化默认日志记录器
// 配置日志输出格式、级别、编码器等参数
// 参数:
//   - config: 日志配置对象
//   - serverName: 服务器名称，用于标识应用
//   - version: 应用版本号
func InitDefaultLogger(config srv_config.LogCfg, serverName string, version string) {
	// 创建生产环境的编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()

	// 自定义时间格式编码器，同时显示格式化时间和毫秒时间戳
	encoderConfig.EncodeTime = zapcore.TimeEncoder(func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(fmt.Sprintf("%v %v", t.Format("2006-01-02 15:04:05.000Z07"), t.UnixMilli()))
	})
	// 根据环境设置日志级别编码器（本地环境使用小写彩色，其他环境使用大写彩色）
	if env.IsLocal() {
		encoderConfig.EncodeLevel = LowercaseColorLevelEncoder
	} else {
		encoderConfig.EncodeLevel = UppercaseColorLevelEncoder
	}
	// 配置其他编码选项
	encoderConfig.LineEnding = zapcore.DefaultLineEnding         // 默认行结束符
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder      // 短格式调用者信息
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder // 字符串格式时长

	// 创建控制台编码器
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	// 设置标准log包的输出格式
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile | log.LstdFlags)
	// 标准库 log 的输出同样按值脱敏（目标仍是默认的标准错误）
	log.SetOutput(redactWriter{w: os.Stderr})

	// 创建日志核心，输出到标准输出；编码器外包一层按值脱敏（secret 组件解析出的凭据明文出现在任何位置都替换为掩码）
	core := zapcore.NewCore(&redactEncoder{Encoder: consoleEncoder}, zapcore.AddSync(os.Stdout), zap.NewAtomicLevelAt(zapcore.Level(config.Level)))

	// 配置日志选项
	opts := []zap.Option{
		zap.AddCaller(),      // 添加调用者信息
		zap.AddCallerSkip(1), // 跳过一层调用栈
	}

	// 如果提供了服务器名称，添加应用名称字段
	if len(serverName) > 0 {
		opts = append(opts, zap.Fields(zap.String("appName", serverName)))
		// 如果提供了版本号，添加版本字段
		if len(version) > 0 {
			opts = append(opts, zap.Fields(zap.String("version", version)))
		}
	}

	// 配置堆栈跟踪选项
	if config.DisableStacktrace {
		stackTrace := zapcore.Level(-1)
		if config.StacktraceLevel == nil {
			stackTrace = zapcore.DPanicLevel
		} else {
			stackTrace = zapcore.Level(*config.StacktraceLevel)
		}
		opts = append(opts, zap.AddStacktrace(stackTrace))
	}

	// 创建默认日志记录器
	defaultLogger = zap.New(core, opts...)
	// 设置第三方库的日志记录器
	setGogoLog(defaultLogger)
	setGinLog(defaultLogger)
}

// InitByDefaultLogger 在默认日志记录器基础上创建新的日志记录器
// 允许添加额外的选项来自定义日志行为
// 参数:
//   - opt: 可变参数，额外的zap选项
//
// 返回值:
//   - *zap.Logger: 配置了额外选项的新日志记录器
func InitByDefaultLogger(opt ...zap.Option) *zap.Logger {
	return defaultLogger.WithOptions(opt...)
}

// _levelToColor 日志级别到颜色的映射表
// 用于在终端中以不同颜色显示不同级别的日志
var _levelToColor = map[zapcore.Level]color{
	zapcore.DebugLevel:  Magenta, // 调试级别使用紫色
	zapcore.InfoLevel:   Blue,    // 信息级别使用蓝色
	zapcore.WarnLevel:   Yellow,  // 警告级别使用黄色
	zapcore.ErrorLevel:  Red,     // 错误级别使用红色
	zapcore.DPanicLevel: Red,     // 开发panic级别使用红色
	zapcore.PanicLevel:  Red,     // panic级别使用红色
	zapcore.FatalLevel:  Red,     // 致命错误级别使用红色
}

// color 终端颜色类型，用于格式化彩色输出
type color string

// add 为字符串添加终端颜色格式
// 参数:
//   - s: 要添加颜色的字符串
//
// 返回值:
//   - string: 带有颜色格式的字符串
func (c color) add(s string) string {
	return fmt.Sprintf(string(c), 0x1B, s, 0x1B)
}

// 终端颜色常量定义，使用ANSI转义序列
const (
	Magenta color = "%c[0;42;35m%v%c[0m" // 紫色（绿底紫字）
	Blue    color = "%c[0;42;30m%v%c[0m" // 蓝色（绿底黑字）
	Yellow  color = "%c[0;43;30m%v%c[0m"
	Red     color = "%c[0;41;30m%v%c[0m"
)

var _unknownLevelColor = Red

func LowercaseColorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	s, ok := _levelToColor[l]
	if !ok {
		s = _unknownLevelColor
	}
	enc.AppendString(s.add(l.CapitalString()))
}
func UppercaseColorLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {

	enc.AppendString(l.CapitalString())
}

func GetFuncName() string {
	pc, file, line, ok := runtime.Caller(2)
	if ok {
		name := runtime.FuncForPC(pc).Name()
		//fmt.Println(file)
		//fmt.Println(line)
		index := strings.LastIndex(file, "/")
		fileName := file[index+1:]
		index = strings.LastIndex(name, ".")
		name = name[index+1:]
		//fmt.Sprintf("%c[1;47;34m%s%c[0m", 0x1B, "[INFO]", 0x1B)
		return fmt.Sprintf("[%v-%v()-%v]", fileName, name, line)

	}
	return ""
}

// PrintPanic 发生panic，在recovery之后使用此函数可以打印panic的堆栈信息
func PrintPanic(err interface{}) {
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	stackInfo := fmt.Sprintf("%s", buf[:n])
	//fmt.Printf("	%s\n\n", stackInfo)
	sb := strings.Builder{}
	s := strings.Split(stackInfo, "\n")

	for i, v := range s {
		if i == 0 {
			sb.WriteString(fmt.Sprintf("%c[0;41;30mpanic %v\n%c", 0x1B, v, 0x1B))
		} else {
			sb.WriteString(fmt.Sprintf("	%v\n", v))
		}
	}
	if err != nil {
		sb.WriteString(fmt.Sprintf("	%+v\n", err))
	}
	fmt.Printf("%v", sb.String())
}

func Info(msg string, fields ...zap.Field) {
	defaultLogger.Info(msg, fields...)
}

func InfoCtx(ctx context.Context, msg string, fields ...zap.Field) {

	fields = append(fields, getZap(ctx)...)
	defaultLogger.Info(msg, fields...)
}

func InfoFmt(ctx context.Context, format string, args ...interface{}) {
	var s string
	if len(args) > 0 && len(format) > 0 {
		s = fmt.Sprintf(format, args...)
	}
	defaultLogger.Info(s)
}

func Warn(msg string, fields ...zap.Field) {
	defaultLogger.Warn(msg, fields...)
}

func WarnCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, getZap(ctx)...)
	defaultLogger.Warn(msg, fields...)
}
func Error(msg string, fields ...zap.Field) {

	for i := 0; i < len(fields); i++ {
		if fields[i].Type == zapcore.ErrorType {
			oerr := fields[i].Interface.(error)
			fields[i].Type = zapcore.StringType
			fields[i].String = oerr.Error()
		}
	}
	defaultLogger.Error(msg, fields...)
}

func ErrorCtx(ctx context.Context, msg string, fields ...zap.Field) {
	for i := 0; i < len(fields); i++ {
		if fields[i].Type == zapcore.ErrorType {
			oerr := fields[i].Interface.(error)
			fields[i].Type = zapcore.StringType
			fields[i].String = oerr.Error()
		}
	}
	fields = append(fields, getZap(ctx)...)
	defaultLogger.Error(msg, fields...)
}

// ErrorWithStack
//
//	@Description: 会打印出stack，支持errors
//	@Author yw 2024-08-06 16:45:35
//	@param msg
//	@param fields
func ErrorWithStack(msg string, fields ...zap.Field) {

	eI := len(fields) - 1
	// z e z z
	// i=0 z e z z =>eI=3
	// i=1 z z z e => eI=2 i=1
	// i=1 z z z e
	for i := 0; i <= eI; i++ {
		if fields[i].Type == zapcore.ErrorType {
			//交换
			fields[i], fields[eI] = fields[eI], fields[i]
			eI--
			i--
		}
	}

	defaultLogger.Error(msg, fields[:eI+1]...)
	for _, v := range fields[eI+1:] {
		fmt.Printf("%+v\n", v.Interface)
	}
}

func Debug(msg string, fields ...zap.Field) {
	defaultLogger.Debug(msg, fields...)
}

func DebugCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, getZap(ctx)...)
	defaultLogger.Debug(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	defaultLogger.Panic(msg, fields...)
}

func PanicCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, getZap(ctx)...)
	defaultLogger.Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	defaultLogger.Fatal(msg, fields...)
}

func FatalCtx(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(fields, getZap(ctx)...)
	defaultLogger.Fatal(msg, fields...)
}

func InfoCtxLogger(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	fields = append(fields, getZap(ctx)...)
	logger.Info(msg, fields...)
}

func WarnCtxLogger(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	fields = append(fields, getZap(ctx)...)
	logger.Warn(msg, fields...)
}

func ErrorCtxLogger(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	for i := 0; i < len(fields); i++ {
		if fields[i].Type == zapcore.ErrorType {
			oerr := fields[i].Interface.(error)
			fields[i].Type = zapcore.StringType
			fields[i].String = oerr.Error()
		}
	}
	fields = append(fields, getZap(ctx)...)
	logger.Error(msg, fields...)
}

func DebugCtxLogger(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	fields = append(fields, getZap(ctx)...)
	logger.Debug(msg, fields...)
}

func PanicCtxLogger(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	fields = append(fields, getZap(ctx)...)
	logger.Panic(msg, fields...)
}

func FatalCtxLogger(ctx context.Context, logger *zap.Logger, msg string, fields ...zap.Field) {
	if logger == nil {
		return
	}
	fields = append(fields, getZap(ctx)...)
	logger.Fatal(msg, fields...)
}
