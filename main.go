package main

import (
	"context"
	"log"
	"log/slog"
	"net/url"
	"os"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sethvargo/go-envconfig"
	"github.com/warden-protocol/wardenprotocol/keychain-sdk"
)

type Config struct {
	ProxyURL     url.URL       `env:"PROXY_URL, default=http://localhost:8080"`
	ProxyTimeout time.Duration `env:"PROXY_TIMEOUT, default=5s"`

	ChainID        string `env:"CHAIN_ID, default=warden"`
	GRPCURL        string `env:"GRPC_URL, default=localhost:9090"`
	GRPCInsecure   bool   `env:"GRPC_INSECURE, default=true"`
	DerivationPath string `env:"DERIVATION_PATH, default=m/44'/118'/0'/0/0"`
	Mnemonic       string `env:"MNEMONIC, default=exclude try nephew main caught favorite tone degree lottery device tissue tent ugly mouse pelican gasp lava flush pen river noise remind balcony emerge"`
	KeychainID     uint64 `env:"KEYCHAIN_ID, default=1"`

	BatchInterval time.Duration `env:"BATCH_INTERVAL, default=8s"`
	BatchSize     int           `env:"BATCH_SIZE, default=7"`
	GasLimit      uint64        `env:"GAS_LIMIT, default=400000"`
	TxTimeout     time.Duration `env:"TX_TIMEOUT, default=120s"`
	TxFee         int64         `env:"TX_FEE, default=400000"`

	LogLevel  slog.Level `env:"LOG_LEVEL, default=info"`
	LogFormat string     `env:"LOG_FORMAT, default=plain"`
}

func main() {
	cfg := readConfig()
	logger := initLogger(cfg)
	client := NewClient(cfg.ProxyURL, cfg.ProxyTimeout)

	app := keychain.NewApp(keychain.Config{
		Logger:         logger,
		ChainID:        cfg.ChainID,
		GRPCURL:        cfg.GRPCURL,
		GRPCInsecure:   cfg.GRPCInsecure,
		DerivationPath: cfg.DerivationPath,
		Mnemonic:       cfg.Mnemonic,
		KeychainID:     cfg.KeychainID,
		GasLimit:       cfg.GasLimit,
		BatchInterval:  cfg.BatchInterval,
		BatchSize:      cfg.BatchSize,
		TxTimeout:      cfg.TxTimeout,
		TxFees:         sdk.NewCoins(sdk.NewCoin("uward", math.NewInt(cfg.TxFee))),
	})

	app.SetKeyRequestHandler(func(w keychain.KeyResponseWriter, req *keychain.KeyRequest) {
		logger := logger.With("id", req.Id)

		res, err := client.requestKey(req)
		if err != nil {
			logger.Error("proxying key request", "err", err)
			_ = w.Reject("internal error")
			return
		}

		if res.Ok {
			if err := w.Fulfil(res.Key); err != nil {
				logger.Error("fulfilling key request", "err", err)
			}
			logger.Info("key request fulfilled")
		} else {
			logger.Error("key request rejected", "reason", res.RejectReason)
			if err := w.Reject(res.RejectReason); err != nil {
				logger.Error("rejecting key request", "err", err)
			}
		}
	})

	app.SetSignRequestHandler(func(w keychain.SignResponseWriter, req *keychain.SignRequest) {
		logger := logger.With("id", req.Id)

		res, err := client.requestSignature(req)
		if err != nil {
			logger.Error("proxying signature request", "err", err)
			_ = w.Reject("internal error")
			return
		}

		if res.Ok {
			if err := w.Fulfil(res.Signature); err != nil {
				logger.Error("fulfilling signature request", "err", err)
			}
			logger.Info("signature request fulfilled")
		} else {
			logger.Error("signature request rejected", "reason", res.RejectReason)
			if err := w.Reject(res.RejectReason); err != nil {
				logger.Error("rejecting signature request", "err", err)
			}
		}
	})

	if err := app.Start(context.Background()); err != nil {
		panic(err)
	}
}

func readConfig() Config {
	var cfg Config
	if err := envconfig.Process(context.Background(), &cfg); err != nil {
		log.Fatal(err)
	}
	return cfg
}

func initLogger(c Config) *slog.Logger {
	out := os.Stderr

	var opts slog.HandlerOptions
	opts.Level = c.LogLevel

	var handler slog.Handler
	switch c.LogFormat {
	case "plain":
		handler = slog.NewTextHandler(out, &opts)
	case "json":
		handler = slog.NewJSONHandler(out, &opts)
	}

	return slog.New(handler)
}
