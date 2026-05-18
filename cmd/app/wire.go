//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/makesalekz/warehouse/internal/biz"
	"github.com/makesalekz/warehouse/internal/conf"
	"github.com/makesalekz/warehouse/internal/data"
	"github.com/makesalekz/warehouse/internal/server"
	"github.com/makesalekz/warehouse/internal/service"
)

func wireApp(*conf.Bootstrap, log.Logger) (*kratos.App, func(), error) {
	panic(wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp))
}
