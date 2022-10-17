package logger

import (
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var glogger *zap.SugaredLogger

var Debug, Info, Error, Fatal func(args ...interface{})
var Errorf, Infof, Fatalf func(template string, args ...interface{})

// InitLogger init

type LoggerConfig struct {
	Filename    string
	MaxSize     int
	MaxAge      int
	MaxBackups  int
	MessageKey  string
	Level       zapcore.Level
	WriteSyncer zapcore.WriteSyncer
	// BJsonFormat  bool
	Encoder func(cfg zapcore.EncoderConfig) zapcore.Encoder
	// Encoder      zapcore.Encoder
	BCaller      bool //是否开启开发模式
	BDevelopment bool //是否开启行号
}

func cstencodeTimeLayout(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	type appendTimeEncoder interface {
		AppendTimeLayout(time.Time, string)
	}

	layout := "2006-01-02 15:04:05.0000"

	t1 := t.UTC().Add(8 * time.Hour)

	if enc, ok := enc.(appendTimeEncoder); ok {
		enc.AppendTimeLayout(t1, layout)
		return
	}

	enc.AppendString(t1.Format(layout))
}

func Initlog(cfg LoggerConfig, options ...zap.Option) {
	if glogger != nil {
		glogger = nil
	}

	nameKey := strings.Split(filepath.Base(cfg.Filename), ".")[0]

	hook := lumberjack.Logger{
		Filename:   cfg.Filename,   // 日志文件路径
		MaxSize:    cfg.MaxSize,    // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: cfg.MaxBackups, // 日志文件最多保存多少个备份
		MaxAge:     cfg.MaxAge,     // 文件最多保存多少天
		Compress:   true,           // 是否压缩
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:    "time",
		LevelKey:   "level",
		NameKey:    nameKey,
		CallerKey:  "linenum",
		MessageKey: cfg.MessageKey,
		// StacktraceKey:  "stacktrace",
		LineEnding:  zapcore.DefaultLineEnding,
		EncodeLevel: zapcore.LowercaseLevelEncoder, // 小写编码器
		// EncodeTime:     zapcore.ISO8601TimeEncoder,     // ISO8601 UTC 时间格式
		EncodeTime:     cstencodeTimeLayout,
		EncodeDuration: zapcore.SecondsDurationEncoder, //
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 全路径编码器
		EncodeName:     zapcore.FullNameEncoder,
	}

	// 设置日志级别
	//	atomicLevel := zap.NewAtomicLevel()
	//	atomicLevel.SetLevel(zap.InfoLevel)

	// conn := udpconn("udp4", &net.UDPAddr{IP: net.ParseIP("192.168.31.125"), Port: 1523})

	var ws zapcore.WriteSyncer
	if cfg.WriteSyncer != nil {
		ws = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&hook), cfg.WriteSyncer)
	} else {
		ws = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(&hook)) // 打印到控制台和文件
	}

	core := zapcore.NewCore(
		cfg.Encoder(encoderConfig),
		ws,
		cfg.Level,
	)

	// 开启开发模式，堆栈跟踪

	//encode := zap.en
	// 设置初始化字段
	//	filed := zap.Fields(zap.String("serviceName", "serviceName"))

	glogger = zap.New(core, options...).Sugar()
	Debug = glogger.Debug
	Errorf = glogger.Errorf
	Error = glogger.Error
	Info = glogger.Info
	Infof = glogger.Infof
	Fatal = glogger.Fatal
	Fatalf = glogger.Fatalf

	// glogger.
	// zap.dai

}
