# Protocol Documentation
<a name="top"></a>

## Table of Contents

- [api/v1/api.proto](#api_v1_api-proto)
    - [AddPeerRequest](#api-v1-AddPeerRequest)
    - [AddPeerResponse](#api-v1-AddPeerResponse)
    - [Group](#api-v1-Group)
    - [PeerInfo](#api-v1-PeerInfo)
    - [PeersResponse](#api-v1-PeersResponse)
    - [PeersResponse.PeersEntry](#api-v1-PeersResponse-PeersEntry)
    - [PluginConfig](#api-v1-PluginConfig)
    - [PluginConfig.ConfigEntry](#api-v1-PluginConfig-ConfigEntry)
    - [PluginConfig.RequiresEntry](#api-v1-PluginConfig-RequiresEntry)
    - [PluginConfigs](#api-v1-PluginConfigs)
    - [PluginID](#api-v1-PluginID)
    - [RemovePeerRequest](#api-v1-RemovePeerRequest)
    - [RemovePeerResponse](#api-v1-RemovePeerResponse)
    - [VersionResponse](#api-v1-VersionResponse)
  
    - [GatewayDAdminAPIService](#api-v1-GatewayDAdminAPIService)
  
- [raft/proto/raft.proto](#raft_proto_raft-proto)
    - [AddPeerRequest](#raft-AddPeerRequest)
    - [AddPeerResponse](#raft-AddPeerResponse)
    - [ForwardApplyRequest](#raft-ForwardApplyRequest)
    - [ForwardApplyResponse](#raft-ForwardApplyResponse)
    - [RemovePeerRequest](#raft-RemovePeerRequest)
    - [RemovePeerResponse](#raft-RemovePeerResponse)
  
    - [RaftService](#raft-RaftService)
  
- [Scalar Value Types](#scalar-value-types)



<a name="api_v1_api-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## api/v1/api.proto



<a name="api-v1-AddPeerRequest"></a>

### AddPeerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| peer_id | [string](#string) |  |  |
| address | [string](#string) |  |  |
| grpc_address | [string](#string) |  |  |






<a name="api-v1-AddPeerResponse"></a>

### AddPeerResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |
| error | [string](#string) |  |  |






<a name="api-v1-Group"></a>

### Group
Group is the object group to filter the global config by.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| group_name | [string](#string) | optional | GroupName is the name of the group. |






<a name="api-v1-PeerInfo"></a>

### PeerInfo



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [string](#string) |  |  |
| address | [string](#string) |  |  |
| state | [string](#string) |  |  |
| is_leader | [bool](#bool) |  |  |
| is_voter | [bool](#bool) |  |  |






<a name="api-v1-PeersResponse"></a>

### PeersResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| peers | [PeersResponse.PeersEntry](#api-v1-PeersResponse-PeersEntry) | repeated |  |






<a name="api-v1-PeersResponse-PeersEntry"></a>

### PeersResponse.PeersEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [PeerInfo](#api-v1-PeerInfo) |  |  |






<a name="api-v1-PluginConfig"></a>

### PluginConfig
PluginConfig is the configuration of the plugin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| id | [PluginID](#api-v1-PluginID) |  | ID is the identifier that uniquely identifies the plugin. |
| description | [string](#string) |  | Description is the description of the plugin. |
| authors | [string](#string) | repeated | Authors is the list of authors of the plugin. |
| license | [string](#string) |  | License is the license of the plugin. |
| project_url | [string](#string) |  | ProjectURL is the project URL of the plugin. |
| config | [PluginConfig.ConfigEntry](#api-v1-PluginConfig-ConfigEntry) | repeated | Config is the internal and external configuration of the plugin. |
| hooks | [int32](#int32) | repeated | Hooks is the list of hooks the plugin attaches to. |
| requires | [PluginConfig.RequiresEntry](#api-v1-PluginConfig-RequiresEntry) | repeated | Requires is the list of plugins the plugin depends on. |
| tags | [string](#string) | repeated | Tags is the list of tags of the plugin. |
| categories | [string](#string) | repeated | Categories is the list of categories of the plugin. |






<a name="api-v1-PluginConfig-ConfigEntry"></a>

### PluginConfig.ConfigEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="api-v1-PluginConfig-RequiresEntry"></a>

### PluginConfig.RequiresEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| key | [string](#string) |  |  |
| value | [string](#string) |  |  |






<a name="api-v1-PluginConfigs"></a>

### PluginConfigs
PluginConfigs is the list of plugin configurations.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| configs | [PluginConfig](#api-v1-PluginConfig) | repeated | Configs is the list of plugin configurations. |






<a name="api-v1-PluginID"></a>

### PluginID
PluginID is the identifier that uniquely identifies the plugin.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  | Name is the name of the plugin. |
| version | [string](#string) |  | Version is the version of the plugin. |
| remote_url | [string](#string) |  | RemoteURL is the remote URL of the plugin. |
| checksum | [string](#string) |  | Checksum is the checksum of the plugin. |






<a name="api-v1-RemovePeerRequest"></a>

### RemovePeerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| peer_id | [string](#string) |  |  |






<a name="api-v1-RemovePeerResponse"></a>

### RemovePeerResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |
| error | [string](#string) |  |  |






<a name="api-v1-VersionResponse"></a>

### VersionResponse
VersionResponse is the response returned by the Version RPC.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| version | [string](#string) |  | Version is the version of the GatewayD. |
| version_info | [string](#string) |  | VersionInfo is the detailed version info of the GatewayD. |





 

 

 


<a name="api-v1-GatewayDAdminAPIService"></a>

### GatewayDAdminAPIService
GatewayDAdminAPIService is the administration API of GatewayD.

| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| Version | [.google.protobuf.Empty](#google-protobuf-Empty) | [VersionResponse](#api-v1-VersionResponse) | Version returns the version of the GatewayD. |
| GetGlobalConfig | [Group](#api-v1-Group) | [.google.protobuf.Struct](#google-protobuf-Struct) | GetGlobalConfig returns the global configuration of the GatewayD. |
| GetPluginConfig | [.google.protobuf.Empty](#google-protobuf-Empty) | [.google.protobuf.Struct](#google-protobuf-Struct) | GetPluginConfig returns the configuration of the specified plugin. |
| GetPlugins | [.google.protobuf.Empty](#google-protobuf-Empty) | [PluginConfigs](#api-v1-PluginConfigs) | GetPlugins returns the list of plugins installed on the GatewayD. |
| GetPools | [.google.protobuf.Empty](#google-protobuf-Empty) | [.google.protobuf.Struct](#google-protobuf-Struct) | GetPools returns the list of pools configured on the GatewayD. |
| GetProxies | [.google.protobuf.Empty](#google-protobuf-Empty) | [.google.protobuf.Struct](#google-protobuf-Struct) | GetProxies returns the list of proxies configured on the GatewayD. |
| GetServers | [.google.protobuf.Empty](#google-protobuf-Empty) | [.google.protobuf.Struct](#google-protobuf-Struct) | GetServers returns the list of servers configured on the GatewayD. |
| GetPeers | [.google.protobuf.Empty](#google-protobuf-Empty) | [.google.protobuf.Struct](#google-protobuf-Struct) | Get information about all peers in the Raft cluster |
| AddPeer | [AddPeerRequest](#api-v1-AddPeerRequest) | [AddPeerResponse](#api-v1-AddPeerResponse) | Add a new peer to the Raft cluster |
| RemovePeer | [RemovePeerRequest](#api-v1-RemovePeerRequest) | [RemovePeerResponse](#api-v1-RemovePeerResponse) | Remove a peer from the Raft cluster |

 



<a name="raft_proto_raft-proto"></a>
<p align="right"><a href="#top">Top</a></p>

## raft/proto/raft.proto



<a name="raft-AddPeerRequest"></a>

### AddPeerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| peer_id | [string](#string) |  |  |
| peer_address | [string](#string) |  |  |
| grpc_address | [string](#string) |  |  |






<a name="raft-AddPeerResponse"></a>

### AddPeerResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |
| error | [string](#string) |  |  |






<a name="raft-ForwardApplyRequest"></a>

### ForwardApplyRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| data | [bytes](#bytes) |  |  |
| timeout_ms | [int64](#int64) |  |  |






<a name="raft-ForwardApplyResponse"></a>

### ForwardApplyResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |
| error | [string](#string) |  |  |






<a name="raft-RemovePeerRequest"></a>

### RemovePeerRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| peer_id | [string](#string) |  |  |






<a name="raft-RemovePeerResponse"></a>

### RemovePeerResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| success | [bool](#bool) |  |  |
| error | [string](#string) |  |  |





 

 

 


<a name="raft-RaftService"></a>

### RaftService


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| ForwardApply | [ForwardApplyRequest](#raft-ForwardApplyRequest) | [ForwardApplyResponse](#raft-ForwardApplyResponse) |  |
| AddPeer | [AddPeerRequest](#raft-AddPeerRequest) | [AddPeerResponse](#raft-AddPeerResponse) |  |
| RemovePeer | [RemovePeerRequest](#raft-RemovePeerRequest) | [RemovePeerResponse](#raft-RemovePeerResponse) |  |

 



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

