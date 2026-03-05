// Copyright (c) 2026 WSO2 LLC. (https://www.wso2.com).
//
// WSO2 LLC. licenses this file to you under the Apache License,
// Version 2.0 (the "License"); you may not use this file except
// in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

# Toke types enum.
public enum TokenType {
    SERVICE_TOKEN = "Service",
    USER_TOKEN = "User"
}

# Enum for permission levels.
public enum PermissionLevel {
    ROBOT = "robot",
    PROJECT = "project",
    SYSTEM = "system"
}

# Enum for resource types.
public enum ResourceType {
    ALL = "*",
    CONFIGURATION = "configuration",
    LABEL = "label",
    LOG = "log",
    LDAP_USER = "ldap-user",
    MEMBER = "member",
    METADATA = "metadata",
    QUOTA = "quota",
    REPOSITORY = "repository",
    TAG_RETENTION = "tag-retention",
    IMMUTABLE_TAG = "immutable-tag",
    ROBOT = "robot",
    NOTIFICATION_POLICY = "notification-policy",
    SCAN = "scan",
    SBOM = "sbom",
    SCANNER = "scanner",
    ARTIFACT = "artifact",
    TAG = "tag",
    ACCESSORY = "accessory",
    ARTIFACT_ADDITION = "artifact-addition",
    ARTIFACT_LABEL = "artifact-label",
    PREHEAT_POLICY = "preheat-policy",
    PREHEAT_INSTANCE = "preheat-instance",
    SELF = "",
    AUDIT_LOG = "audit-log",
    CATALOG = "catalog",
    PROJECT = "project",
    USER = "user",
    USER_GROUP = "user-group",
    REGISTRY = "registry",
    REPLICATION = "replication",
    DISTRIBUTION = "distribution",
    GARBAGE_COLLECTION = "garbage-collection",
    REPLICATION_ADAPTER = "replication-adapter",
    REPLICATION_POLICY = "replication-policy",
    SCAN_ALL = "scan-all",
    SYSTEM_VOLUMES = "system-volumes",
    PURGE_AUDIT_LOG = "purge-audit",
    EXPORT_CVE = "export-cve",
    JOBSERVICE_MONITOR = "jobservice-monitor",
    SECURITY_HUB = "security-hub"
}

# Enum for access actions.
public enum AccessAction {
    ALL = "*",
    PULL = "pull",
    PUSH = "push",
    CREATE = "create",
    READ = "read",
    UPDATE = "update",
    DELETE = "delete",
    LIST = "list",
    OPERATE = "operate",
    SCANNER_PULL = "scanner-pull",
    STOP = "stop"
}
