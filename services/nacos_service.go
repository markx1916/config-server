package services

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/yourusername/nacos-config-center/config" // Adjust to your module path
	"github.com/yourusername/nacos-config-center/models" // Adjust to your module path
	"github.com/yourusername/nacos-config-center/utils"  // Adjust to your module path
	"go.uber.org/zap"
)

var (
	nacosClients = make(map[string]config_client.IConfigClient)
	mu           sync.Mutex
)

// getNacosClient creates or retrieves a cached Nacos config client for a given instance.
func getNacosClient(instance *models.NacosInstance) (config_client.IConfigClient, error) {
	mu.Lock()
	defer mu.Unlock()

	clientKey := fmt.Sprintf("%s-%s", instance.ServerAddr, instance.NamespaceID)
	if client, ok := nacosClients[clientKey]; ok {
		// Potentially add a health check for the cached client here if the SDK supports it
		utils.Logger.Debug("Using cached Nacos client", zap.String("key", clientKey))
		return client, nil
	}

	// Parse ServerAddr: "host1:port1,host2:port2" into []constant.ServerConfig
	serverConfigs := []constant.ServerConfig{}
	addrs := strings.Split(instance.ServerAddr, ",")
	for _, addr := range addrs {
		parts := strings.Split(strings.TrimSpace(addr), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid server address format: %s", addr)
		}
		port, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid port in server address: %s", parts[1])
		}
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(parts[0], port))
	}

	if len(serverConfigs) == 0 {
		return nil, fmt.Errorf("no valid Nacos server addresses configured for instance: %s", instance.Name)
	}

	// ClientConfig
	// Use global Nacos settings from config.yaml as fallback/base for timeout, scheme, contextPath
	// The instance specific details (serverAddr, namespace, auth) override these.
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(instance.NamespaceID),
		constant.WithTimeoutMs(config.AppConfig.Nacos.TimeoutMs), // From global config
		constant.WithBeatInterval(2*1000), //ms
		constant.WithNotLoadCacheAtStart(true),
		constant.WithUsername(instance.Username),
		constant.WithPassword(instance.Password),
		// Scheme and ContextPath might need to be part of NacosInstance model if they vary per instance
		// For now, assuming they are somewhat standard or can be derived.
		// constant.WithScheme(config.AppConfig.Nacos.Scheme),
		// constant.WithContextPath(config.AppConfig.Nacos.ContextPath),
	)
	// SDK v2 recommends using LogDir and CacheDir for log and cache files.
    // cc.LogDir = "/tmp/nacos/log"
    // cc.CacheDir = "/tmp/nacos/cache"


	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: serverConfigs,
		},
	)

	if err != nil {
		utils.Logger.Error("Failed to create Nacos client",
			zap.String("instanceName", instance.Name),
			zap.String("serverAddr", instance.ServerAddr),
			zap.String("namespaceId", instance.NamespaceID),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create Nacos client for instance %s: %w", instance.Name, err)
	}

	nacosClients[clientKey] = client
	utils.Logger.Info("Nacos client created and cached", zap.String("key", clientKey))
	return client, nil
}

// PublishConfig publishes a configuration to a Nacos instance.
// For gray publishing, Nacos SDK v2 for Go might require specific parameters or it might be handled by Nacos server logic
// based on how it interprets a normal publish call (e.g. if beta publishing is enabled on server).
// The `PublishConfig` method in SDK v2 doesn't have explicit 'beta' or 'gray' flags in its primary signature.
// Gray publishing often involves publishing with a specific `betaIps` parameter, which can be passed in `vo.ConfigParam`.
func PublishConfig(instance *models.NacosInstance, config *models.Configuration, content string, deploymentType string, betaIps string) (bool, error) {
	client, err := getNacosClient(instance)
	if err != nil {
		return false, err
	}

	params := vo.ConfigParam{
		DataId:  config.DataID,
		Group:   config.GroupName,
		Content: content,
		Type:    vo.ConfigType(config.Format), // Ensure config.Format matches one of Nacos's types (text, json, xml, yaml, html, properties)
	}

	// For Nacos SDK v2, if you're doing a beta/gray publish, you might set BetaIps.
	// This depends on how Nacos server handles it and if the SDK exposes this directly
	// in PublishConfig or if it's a separate method/parameter.
	// The standard PublishConfig in v2.x takes `vo.ConfigParam` which includes `BetaIps`.
	if deploymentType == "GRAY" && betaIps != "" {
		params.BetaIps = betaIps // Comma-separated IP list
		utils.Logger.Info("Attempting GRAY publish", zap.String("dataId", config.DataID), zap.String("group", config.GroupName), zap.String("betaIps", betaIps))
	} else {
		utils.Logger.Info("Attempting FULL publish", zap.String("dataId", config.DataID), zap.String("group", config.GroupName))
	}


	success, err := client.PublishConfig(params)
	if err != nil {
		utils.Logger.Error("Failed to publish config to Nacos",
			zap.String("instanceName", instance.Name),
			zap.String("dataId", config.DataID),
			zap.String("group", config.GroupName),
			zap.Error(err),
		)
		return false, fmt.Errorf("Nacos PublishConfig failed: %w", err)
	}
	if !success {
		utils.Logger.Warn("Nacos PublishConfig returned false (not success)",
			zap.String("instanceName", instance.Name),
			zap.String("dataId", config.DataID),
			zap.String("group", config.GroupName),
		)
		// This case (success==false, err==nil) might indicate a known non-error failure by Nacos.
		return false, fmt.Errorf("Nacos PublishConfig was not successful (returned false by SDK)")
	}

	utils.Logger.Info("Successfully published config to Nacos",
		zap.String("instanceName", instance.Name),
		zap.String("dataId", config.DataID),
		zap.String("group", config.GroupName),
		zap.String("type", deploymentType),
	)
	return success, nil
}

// GetConfig retrieves a configuration from a Nacos instance.
// Useful for validation or checking current state in Nacos if needed.
func GetNacosConfig(instance *models.NacosInstance, dataId, group string) (string, error) {
	client, err := getNacosClient(instance)
	if err != nil {
		return "", err
	}

	content, err := client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})

	if err != nil {
		// Nacos SDK might return an error if config is not found.
		// It's important to distinguish "not found" from other errors if possible.
		// The SDK error messages or types would need to be checked for this.
		utils.Logger.Error("Failed to get config from Nacos",
			zap.String("instanceName", instance.Name),
			zap.String("dataId", dataId),
			zap.String("group", group),
			zap.Error(err),
		)
		return "", fmt.Errorf("Nacos GetConfig failed: %w", err)
	}
	return content, nil
}
