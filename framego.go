package framego

import "github.com/frame-go/framego/appmgr"

type AppInfo = appmgr.AppInfo
type App = appmgr.App
type Service = appmgr.Service
type DatabaseManager = appmgr.DatabaseManager

func NewApp(info *AppInfo) App {
	return appmgr.NewApp(info)
}
