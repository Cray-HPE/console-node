// MIT License
//
// (C) Copyright [2020-2021] Hewlett Packard Enterprise Development LP
//
// Permission is hereby granted, free of charge, to any person obtaining a
// copy of this software and associated documentation files (the "Software"),
// to deal in the Software without restriction, including without limitation
// the rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of the Software, and to permit persons to whom the
// Software is furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included
// in all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR
// OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE,
// ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR
// OTHER DEALINGS IN THE SOFTWARE.

package main

import (

	"os"
	"strings"

	"github.com/namsral/flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/Cray-HPE/hmcollector"

)

const NumWorkers = 30
const EndpointRefreshInterval = 5



var (
	kafkaBrokersConfigFile = flag.String("kafka_brokers_config", "/configs/kafka_brokers.json",
		"Path to the configuration file containing all of the Kafka brokers this collector should produce to.")

	serviceName  string
	kafkaBrokers []*hmcollector.KafkaBroker

	atomicLevel zap.AtomicLevel
	logger      *zap.Logger
	Running = true
)


func SetupLogging() {
	logLevel := os.Getenv("LOG_LEVEL")
	logLevel = strings.ToUpper(logLevel)

	atomicLevel = zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()
	logger = zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.Lock(os.Stdout),
		atomicLevel,
	))

	switch logLevel {
	case "DEBUG":
		atomicLevel.SetLevel(zap.DebugLevel)
	case "INFO":
		atomicLevel.SetLevel(zap.InfoLevel)
	case "WARN":
		atomicLevel.SetLevel(zap.WarnLevel)
	case "ERROR":
		atomicLevel.SetLevel(zap.ErrorLevel)
	case "FATAL":
		atomicLevel.SetLevel(zap.FatalLevel)
	case "PANIC":
		atomicLevel.SetLevel(zap.PanicLevel)
	default:
		atomicLevel.SetLevel(zap.InfoLevel)
	}
}

// func main() {
// 	var err error

// 	SetupLogging()




// 	// NOTE: will wait within this function forever if kafka doesn't connect
// 	setupKafka()


// 	// Close the connection to Kafka to make sure any buffered data gets flushed.
// 	defer func() {
// 		for idx := range kafkaBrokers {
// 			thisBroker := kafkaBrokers[idx]

// 			// This call to Flush is given a maximum timeout of 15 seconds (which is entirely arbitrary and should
// 			// never take that long). It's very likely this will return almost immediately in most cases.
// 			abandonedMessages := thisBroker.KafkaProducer.Flush(15 * 1000)
// 			logger.Info("Closed connection with Kafka broker.",
// 				zap.Any("broker", thisBroker),
// 				zap.Int("abandonedMessages", abandonedMessages))
// 		}

// 	}()

// }

