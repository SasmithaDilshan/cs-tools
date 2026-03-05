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

# Create a registry token.
#
# + payload - Robot account creation payload
# + return - Created registry token or an error
public isolated function createRegistryToken(RegistryTokenCreatePayload payload) returns RegistryTokenResponse|error {
    return registryTokensClient->/robot\-accounts.post(payload);
}

# Search registry tokens.
#
# + payload - Registry token search payload
# + return - List of registry tokens matching the search criteria or an error
public isolated function searchRegistryTokens(RegistryTokenSearchPayload payload)
    returns RegistryTokenResponse[]|error {

    return registryTokensClient->/robot\-accounts/search.post(payload);
}

# Delete a registry token.
#
# + tokenId - ID of the registry token to be deleted
# + return - Success message or an error
public isolated function deleteRegistryToken(string tokenId) returns error? {
    return registryTokensClient->/robot\-accounts/[tokenId].delete();
}

# Regenerate a registry token.
#
# + tokenId - ID of the registry token to be regenerated
# + return - Regenerated registry token or an error
public isolated function regenerateRegistryToken(string tokenId) returns RegistryTokenResponse|error {
    return registryTokensClient->/robot\-accounts/[tokenId]/regenerate.post({});
}
