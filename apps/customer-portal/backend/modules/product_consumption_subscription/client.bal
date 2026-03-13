// Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com). All Rights Reserved.
//
// This software is the property of WSO2 LLC. and its suppliers, if any.
// Dissemination of any information or reproduction of any material contained
// herein in any form is strictly forbidden, unless permitted by WSO2 expressly.
// You may not alter or remove any copyright or other notice from copies of this content.
import ballerina/http;

configurable string productConsumptionBaseUrl = ?;
configurable ClientAuthConfig clientAuthConfig = ?;

@display {
    label: "Product Consumption Subscription",
    id: "product-consumption-subscription"
}
final http:Client productConsumptionClient = check new (productConsumptionBaseUrl, {
    auth: {...clientAuthConfig},
    http1Settings: {
        keepAlive: http:KEEPALIVE_NEVER
    }
});
