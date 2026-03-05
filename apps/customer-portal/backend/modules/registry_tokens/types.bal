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
import ballerina/constraint;

# [Configurable] Client credentials grant type oauth2 configuration.
type ClientCredentialsOauth2Config record {|
    # OAuth2 token endpoint
    string tokenUrl;
    # OAuth2 client ID
    string clientId;
    # OAuth2 client secret
    string clientSecret;
    # OAuth2 scopes
    string[] scopes = [];
|};


# Registry token creation payload.
public type RegistryTokenCreatePayload record {|
    # Customer account name
    string accountName;
    # Customer project key
    string projectKey;
    # Registry token name (provided by the user)
    @constraint:String {
        pattern: {
            value: re `^[a-zA-Z0-9\-]+$`,
            message: "Name can only contain alphanumeric characters and dashes"
        }
    }
    string robotName;
    # ServiceNow Account ID
    string snAccountId;
    # ServiceNow Project ID
    string snProjectId;
    # Token Type
    TokenType tokenType;
    # Created for user email
    string createdFor;
    # Created by user email
    string createdBy;
|};

# Subscription sync payload.
public type SubscriptionSyncPayload record {|
    # Opportunity ID list
    string[] opportunityIds;
    # Email of the contact (optional)
    string contactEmail?;
|};

# Partner sync payload.
public type PartnerSyncPayload record {|
    # List of partner emails to be synced
    @constraint:Array {
        minLength: {
            value: 1,
            message: "At least one partner email must be provided"
        }
    }
    string[] emails;
|};

# Response of the created registry token.
public type RegistryTokenResponse record {|
    # Registry token Secret
    string secret;
    json...;
|};

# Registry token search payload.
public type RegistryTokenSearchPayload record {|
    # ServiceNow Account ID
    string snAccountId;
    # ServiceNow Project ID
    string snProjectId;
    # User email
    string userEmail?;
    # Is admin user
    boolean isAdmin = false;
|};

# Registry token configuration.
public type RegistryToken record {|
    # ID
    int id?;
    # Name
    string name;
    # Display name (name provide by the user)
    string displayName?;
    # Description
    string description;
    # Creation time of the registry token
    string creationTime?;
    # Token type
    TokenType tokenType?;
    # Created for
    string createdFor?;
    # Created by
    string createdBy?;
    # Expiry timestamp of the registry token
    int expiresAt?;
    # Permission level
    PermissionLevel level = SYSTEM;
    # Whether the registry token is disabled
    boolean disable = false;
    # Duration for the registry token (-1 for unlimited)
    int duration = -1;
    # Permissions granted to the registry token
    Permission[] permissions;
    json...;
|};

# Permission configuration for registry token.
public type Permission record {|
    # Kind of permission (e.g., "project")
    string kind = "project";
    # Namespace for the permission
    string namespace;
    # Access rights for this permission
    Access[] access = [];
|};

# Access configuration for registry token permissions.
public type Access record {|
    # Resource type
    ResourceType 'resource = REPOSITORY;
    # Action allowed on the resource
    AccessAction action;
|};
