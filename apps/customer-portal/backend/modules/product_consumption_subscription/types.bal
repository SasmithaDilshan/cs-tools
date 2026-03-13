# Client authentication optional parameters structure.
public type OptionalParams record {|
    # Organization Handle
    string orgHandle?;
|};

# Client authentication configuration structure.
public type ClientAuthConfig record {|
    # Token URL
    string tokenUrl;
    # Client ID
    string clientId;
    # Client Secret
    string clientSecret;
|};

# ServiceNow response structure.
public type Result record {|
    # Response message
    string message?;
    # ServiceNow system identifier
    string sys_id?;
    # Choreo application ID
    string applicationId?;
    Result1 result;
|};

# ServiceNow response structure.
public type Result1 record {|
    # Response message
    int status;
    # Choreo application ID
    string applicationId?;
    json ...;
|};

# Project status request payload structure (inbound to this service).
public type ProjectStatusRequest record {
    # Email of the user
    string email;
    # Unique deployment identifier
    string deploymentId;
    # Unique project identifier
    string projectId;
};

# Choreo application creation response structure.
public type ApplicationCreateResponse record {
    # Application ID
    string id;
};

# Choreo credentials generation response structure.
public type CredentialsResponse record {
    # OAuth2 consumer key
    string consumerKey;
    # OAuth2 consumer secret
    string consumerSecret;
};

# Choreo secret keys generation response structure.
public type SecretKeysResponse record {
    # Primary subscription secret key
    string primarySecretKey;
    # Secondary subscription secret key
    string secondarySecretKey;
};

# Subscription data within the license response.
public type SubscriptionData record {|
    # Deployment identifier
    string deploymentId;
    # Deployment name
    string deploymentName;
    # Subscription key
    string subscriptionKey;
    # OAuth2 client ID
    string clientId;
    # OAuth2 client secret
    string clientSecret;
    # Subscription secrets
    string secrets;
|};

# License details within the license response.
public type License record {|
    # Subscription data
    SubscriptionData subscriptionData;
    # Response signature
    string signature;
|};

# License result object nested under 'result'.
public type LicenseResult record {|
    # Success flag
    boolean success;
    # Project sys_id
    string project_id;
    # Deployment sys_id
    string deployment_id;
    # User email
    string email;
    # License details
    License license;
|};

# License download response structure.
public type LicenseResponse record {|
    # Result object
    LicenseResult result;
|};