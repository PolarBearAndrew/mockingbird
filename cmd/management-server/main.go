package main

import (
	"flag"
	"github.com/lab-envoy/pkg/dao"
	"github.com/lab-envoy/pkg/service"
	"github.com/lab-envoy/pkg/utils"
	"log"
	"sync"
)

var (
	logger         utils.Logger
	port           uint
	nodeID         string
	routesConfFile string
)

func init() {
	logger = utils.Logger{}
	flag.BoolVar(&logger.Debug, "debug", false, "Enable xDS server debug logging")
	flag.UintVar(&port, "port", 4000, "xDS management server port")
	flag.StringVar(&nodeID, "nodeID", "mockingbird-default-id", "Node ID")
	flag.StringVar(&routesConfFile, "routesConfFile", "/src/mockingbird.config.json", "Routes config file path")
}

func main() {

	flag.Parse()

	snapshotInternalMemoryDao := dao.NewInternalMemorySnapshotDao()
	snapshotCtrl := service.NewSnapshotController(nodeID, &snapshotInternalMemoryDao, logger)

	err := snapshotCtrl.Init(service.InitOpt{
		InitFile: routesConfFile,
	})

	if err != nil {
		logger.Errorf("snapshotCtrl.Init failed", err)
	}

	if err := snapshotCtrl.RefreshSnapshot(); err != nil {
		logger.Errorf("snapshotCtrl refresh snapshot failed", err)
	}

	managementServiceConfig := &service.EnvoyManagementServerConfig{
		Port:               port,
		Logger:             &logger,
		SnapshotController: &snapshotCtrl,
	}

	opConf := &service.OperationServerConf{
		Port: 3000,
	}

	opBase := &service.OperationServerBase{
		Logger:       &logger,
		SnapshotCtrl: &snapshotCtrl,
	}

	RunServers(managementServiceConfig, opConf, opBase)
}

func RunServers(
	m *service.EnvoyManagementServerConfig,
	opConf *service.OperationServerConf,
	opBase *service.OperationServerBase,
) {

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		log.Printf("management server will listen on %d\n", m.Port)
		if err, _ := service.NewGRCPManagementServer(m); err != nil {
			log.Println(err)
		}
	}()

	go func() {
		defer wg.Done()

		log.Printf("operation server will listen HTTP/1.1 on %d\n", opConf.Port)
		if err := service.NewHttpOperationServer(opConf, opBase).ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	wg.Wait()
}