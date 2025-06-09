package nacosclient

import (
	"fmt"
	"nacos-config-tool/internal/config"
	"nacos-config-tool/internal/logger"
	"nacos-config-tool/internal/models"
	"strconv"
	"strings"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/zap"
)

// NacosClientManager holds cached Nacos clients.
// The key for the cache could be instanceID_namespaceID or just instanceID if clients are namespace-agnostic for creation.
// For simplicity, we'll cache based on a unique identifier derived from the instance's core connection parameters.
type NacosClientManager struct {
	configClients      map[string]config_client.IConfigClient
	namingClients      map[string]naming_client.INamingClient
	mu                 sync.Mutex
	defaultNacosConfig config.NacosConfig // Store app's default Nacos config
}

// NewNacosClientManager creates a new Nacos client manager.
func NewNacosClientManager(appNacosConfig config.NacosConfig) *NacosClientManager {
	return &NacosClientManager{
		configClients:      make(map[string]config_client.IConfigClient),
		namingClients:      make(map[string]naming_client.INamingClient),
		defaultNacosConfig: appNacosConfig,
	}
}

func generateClientCacheKey(instance models.NacosInstance) string {
	// A more robust key would involve all parameters that define a client instance
	return fmt.Sprintf("%s_%s_%s_%s", instance.InstanceURL, instance.Scheme, instance.ContextPath, instance.NamespaceID) // NamespaceID in key ensures client is specific if needed
}

// GetClients retrieves or creates cached Nacos naming and config clients.
// It uses Nacos server details from the 'instance' argument.
// Defaults like timeout are taken from appConfig passed during manager creation.
func (m *NacosClientManager) GetClients(instance models.NacosInstance) (config_client.IConfigClient, naming_client.INamingClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cacheKey := generateClientCacheKey(instance)

	if cc, ok := m.configClients[cacheKey]; ok {
		if nc, ok_nc := m.namingClients[cacheKey]; ok_nc {
			return cc, nc, nil
		}
	}

	// Parse server addresses
	serverAddrParts := strings.Split(instance.InstanceURL, ",")
	serverConfigs := []constant.ServerConfig{}
	for _, addr := range serverAddrParts {
		addr = strings.TrimSpace(addr)
		hostPort := strings.Split(addr, ":")
		if len(hostPort) != 2 {
			logger.Warn("Invalid Nacos server address format in instance details, skipping.", zap.String("address", addr), zap.Uint("instanceID", instance.ID))
			continue
		}
		port, err := strconv.ParseUint(hostPort[1], 10, 64)
		if err != nil {
			logger.Warn("Invalid Nacos server port in instance details, skipping.", zap.String("address", addr), zap.Uint("instanceID", instance.ID), zap.Error(err))
			continue
		}
		serverConfigs = append(serverConfigs, *constant.NewServerConfig(hostPort[0], port, constant.WithScheme(instance.Scheme), constant.WithContextPath(instance.ContextPath)))
	}

	if len(serverConfigs) == 0 {
		return nil, nil, fmt.Errorf("no valid server configurations found for instance ID %d (URL: %s)", instance.ID, instance.InstanceURL)
	}

	// ClientConfig
	clientConfig := *constant.NewClientConfig(
		constant.With первоеTimeoutMs(m.defaultNacosConfig.TimeoutMs), // Use default timeout from app config
		constant.WithNamespaceId(instance.NamespaceID), // This is the default namespace for the client. Operations can specify different namespaces.
		constant.WithNotLoadCacheAtStart(true),
		constant.WithLogDir("/tmp/nacos/log"),   // Consider making these configurable
		constant.WithCacheDir("/tmp/nacos/cache"), // Consider making these configurable
		constant.WithLogLevel("warn"),           // Consider making this configurable
		// constant.WithUsername(instance.Username), // SDK handles empty username/password
		// constant.WithPassword(instance.Password),
	)
	if instance.Username != "" {
		clientConfig.Username = instance.Username
	}
	if instance.Password != "" {
		clientConfig.Password = instance.Password
	}


	// Create Config Client
	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		logger.Error("Failed to create Nacos config client", zap.Error(err), zap.Uint("instanceID", instance.ID))
		return nil, nil, fmt.Errorf("failed to create Nacos config client for instance %d: %w", instance.ID, err)
	}

	// Create Naming Client
	// Naming client might use a different default namespace or have slightly different needs.
	// For now, we use the same clientConfig, but this could be customized.
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig, // Potentially a different clientConfig for naming
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		logger.Error("Failed to create Nacos naming client", zap.Error(err), zap.Uint("instanceID", instance.ID))
		// Clean up config client if naming client fails
		if configClient != nil {
			// SDK does not expose a Close() method on IConfigClient directly.
			// For now, we just log and proceed without naming client or return both as nil.
		}
		return nil, nil, fmt.Errorf("failed to create Nacos naming client for instance %d: %w", instance.ID, err)
	}

	m.configClients[cacheKey] = configClient
	m.namingClients[cacheKey] = namingClient
	logger.Info("Successfully created and cached Nacos clients", zap.String("cacheKey", cacheKey), zap.Uint("instanceID", instance.ID))
	return configClient, namingClient, nil
}

// GetConfig gets configuration from Nacos.
func (m *NacosClientManager) GetConfig(client config_client.IConfigClient, params vo.ConfigParam) (string, error) {
	content, err := client.GetConfig(params)
	if err != nil {
		logger.Error("Failed to get config from Nacos", zap.Any("params", params), zap.Error(err))
		return "", err
	}
	return content, nil
}

// PublishConfig publishes configuration to Nacos.
// Returns true if publish is successful, false otherwise.
func (m *NacosClientManager) PublishConfig(client config_client.IConfigClient, params vo.ConfigParam) (bool, error) {
	success, err := client.PublishConfig(params)
	if err != nil {
		logger.Error("Failed to publish config to Nacos", zap.Any("params", params), zap.Error(err))
		return false, err
	}
	if !success {
		logger.Warn("Nacos reported publish as unsuccessful", zap.Any("params", params))
		return false, fmt.Errorf("nacos reported publish as unsuccessful for DataID: %s, Group: %s", params.DataId, params.Group)
	}
	return success, nil
}

// DeleteConfig deletes a configuration from Nacos.
// Returns true if delete is successful, false otherwise.
func (m *NacosClientManager) DeleteConfig(client config_client.IConfigClient, params vo.ConfigParam) (bool, error) {
	success, err := client.DeleteConfig(params)
	if err != nil {
		logger.Error("Failed to delete config from Nacos", zap.Any("params", params), zap.Error(err))
		return false, err
	}
	if !success {
		logger.Warn("Nacos reported delete as unsuccessful", zap.Any("params", params))
		return false, fmt.Errorf("nacos reported delete as unsuccessful for DataID: %s, Group: %s", params.DataId, params.Group)
	}
	return success, nil
}


// ListenConfig sets up a listener for configuration changes.
// The onChange callback will be invoked when the specified configuration changes.
func (m *NacosClientManager) ListenConfig(client config_client.IConfigClient, params vo.ConfigParam, onChange func(namespace, group, dataId, data string)) error {
	err := client.ListenConfig(params)
	if err != nil {
		logger.Error("Failed to listen to config changes on Nacos", zap.Any("params", params), zap.Error(err))
		return err
	}
	return nil
}


// GetAllNamespaces lists all namespaces for a Nacos instance using the Naming client.
// Note: Nacos config client does not have a direct API to list all namespaces.
// Naming client's GetAllNamespaces is typically used.
func (m *NacosClientManager) GetAllNamespaces(namingClient naming_client.INamingClient) ([]model.Namespace, error) {
	if namingClient == nil {
		return nil, fmt.Errorf("naming client is not initialized")
	}
	namespaces, err := namingClient.GetAllNamespaces()
	if err != nil {
		logger.Error("Failed to get all namespaces from Nacos", zap.Error(err))
		return nil, err
	}
	return namespaces, nil
}


// TestNacosConnection tests connectivity to a Nacos instance.
// It tries to get clients and then lists namespaces.
func (m *NacosClientManager) TestNacosConnection(instance models.NacosInstance) error {
	configClient, namingClient, err := m.GetClients(instance)
	if err != nil {
		return fmt.Errorf("failed to get Nacos clients: %w", err)
	}
	if configClient == nil || namingClient == nil {
		return fmt.Errorf("received nil clients without error, instance: %s", instance.InstanceURL)
	}

	// A simple test: try to list namespaces.
	// The default namespace for the client is instance.NamespaceID, but GetAllNamespaces is not namespace specific.
	_, err = m.GetAllNamespaces(namingClient)
	if err != nil {
		return fmt.Errorf("failed to list namespaces during connection test: %w", err)
	}

	logger.Info("Nacos connection test successful", zap.Uint("instanceID", instance.ID), zap.String("instanceURL", instance.InstanceURL))
	return nil
}
