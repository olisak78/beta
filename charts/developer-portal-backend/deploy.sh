#!/bin/bash

# Deploy script for Developer Portal Backend Helm Chart
# Usage: ./deploy.sh <dev|prod>
# Example: ./deploy.sh prod
# Example: ./deploy.sh dev
# For custom values: ./deploy.sh dev -f values-custom.yaml
#
# Environment-specific GitHub OAuth credentials:
# - dev:  Uses GITHUB_TOOLS_APP_CLIENT_ID_DEV and GITHUB_TOOLS_APP_CLIENT_SECRET_DEV
# - prod: Uses GITHUB_TOOLS_APP_CLIENT_ID_PROD and GITHUB_TOOLS_APP_CLIENT_SECRET_PROD
# - local development: Uses GITHUB_TOOLS_APP_CLIENT_ID (no suffix)
#
# Optional AI Core credentials:
# - Set AI_CORE_CREDENTIALS in .env as JSON array with ALL required fields:
#   export AI_CORE_CREDENTIALS='[{"team":"team-name","clientId":"id","clientSecret":"secret","oauthUrl":"https://...","apiUrl":"https://...","resourceGroup":"default"}]'

set -e

# Cleanup function for temporary files
cleanup() {
    if [ ! -z "$AI_CORE_VALUES_FILE" ] && [ -f "$AI_CORE_VALUES_FILE" ]; then
        rm -f "$AI_CORE_VALUES_FILE"
    fi
}
trap cleanup EXIT

# Environment parameter (dev or prod) - REQUIRED
DEPLOY_ENVIRONMENT=$1

# Validate environment parameter
if [[ -z "$DEPLOY_ENVIRONMENT" ]]; then
    echo "❌ ERROR: Environment parameter is required"
    echo "Usage: ./deploy.sh <dev|prod>"
    echo "Example: ./deploy.sh dev"
    echo "Example: ./deploy.sh prod"
    exit 1
fi

if [[ "$DEPLOY_ENVIRONMENT" != "dev" && "$DEPLOY_ENVIRONMENT" != "prod" ]]; then
    echo "❌ ERROR: Environment parameter must be 'dev' or 'prod'"
    echo "Usage: ./deploy.sh <dev|prod>"
    echo "Example: ./deploy.sh dev"
    echo "Example: ./deploy.sh prod"
    exit 1
fi

NAMESPACE="developer-portal"
RELEASE_NAME="developer-portal-backend"
CHART_DIR="."
VALUES_FILE="values.yaml"

# Set ingress host based on environment
BASE_HOST="backend.developer-portal.cfs.c.eu-de-2.cloud.sap"
if [ "$DEPLOY_ENVIRONMENT" == "dev" ]; then
    INGRESS_HOST="dev.$BASE_HOST"
else
    INGRESS_HOST="$BASE_HOST"
fi

# Set auth redirect URL to match ingress host
AUTH_REDIRECT_URL="https://$INGRESS_HOST"

# Check for additional -f flag for custom values file
shift || true  # Remove first argument (environment)
EXTRA_ARGS="$@"  # Capture any remaining arguments (like -f values-custom.yaml)

echo "🚀 Deploying Developer Portal Backend"
echo "   Environment: $DEPLOY_ENVIRONMENT"
echo "   Namespace: $NAMESPACE"
echo "   Ingress Host: $INGRESS_HOST"
echo "   Values File: $VALUES_FILE"
if [ "$DEPLOY_ENVIRONMENT" == "dev" ]; then
    echo "   GitHub OAuth: Using *_DEV environment variables"
else
    echo "   GitHub OAuth: Using *_PROD environment variables"
fi
if [ ! -z "$EXTRA_ARGS" ]; then
    echo "   Extra Args: $EXTRA_ARGS"
fi
echo ""
ENV_PATH="./../../.env"

# Try to source .env file first if it exists
if [ -f "$ENV_PATH" ]; then
    echo "📥 Loading environment variables from .env..."
    source "$ENV_PATH"
    echo "✅ Loaded .env file"
    echo ""
fi

# Map environment-specific credentials to standard variable names
# This allows dev/prod to use separate secrets while keeping local development simple
# NOTE: This must happen AFTER sourcing .env file
if [ "$DEPLOY_ENVIRONMENT" == "dev" ]; then
    # Use DEV-suffixed variables for dev environment
    GITHUB_TOOLS_APP_CLIENT_ID="${GITHUB_TOOLS_APP_CLIENT_ID_DEV}"
    GITHUB_TOOLS_APP_CLIENT_SECRET="${GITHUB_TOOLS_APP_CLIENT_SECRET_DEV}"
    GITHUB_WDF_APP_CLIENT_ID="${GITHUB_WDF_APP_CLIENT_ID_DEV}"
    GITHUB_WDF_APP_CLIENT_SECRET="${GITHUB_WDF_APP_CLIENT_SECRET_DEV}"
    JWT_SECRET="${JWT_SECRET_DEV}"
    TOKEN_SECRET="${TOKEN_SECRET_DEV}"
    OAUTH_ENCRYPTION_KEY="${OAUTH_ENCRYPTION_KEY_DEV}"
else
    # Use PROD-suffixed variables for prod environment
    GITHUB_TOOLS_APP_CLIENT_ID="${GITHUB_TOOLS_APP_CLIENT_ID_PROD}"
    GITHUB_TOOLS_APP_CLIENT_SECRET="${GITHUB_TOOLS_APP_CLIENT_SECRET_PROD}"
    GITHUB_WDF_APP_CLIENT_ID="${GITHUB_WDF_APP_CLIENT_ID_PROD}"
    GITHUB_WDF_APP_CLIENT_SECRET="${GITHUB_WDF_APP_CLIENT_SECRET_PROD}"
    JWT_SECRET="${JWT_SECRET_PROD}"
    TOKEN_SECRET="${TOKEN_SECRET_PROD}"
    OAUTH_ENCRYPTION_KEY="${OAUTH_ENCRYPTION_KEY_PROD}"
fi

echo "📝 Mapped environment-specific credentials for $DEPLOY_ENVIRONMENT environment"
echo ""

# Function to check for missing required variables
check_required_vars() {
    local missing=()

    [ -z "$JWT_SECRET" ] && missing+=("JWT_SECRET")
    [ -z "$TOKEN_SECRET" ] && missing+=("TOKEN_SECRET")
    [ -z "$DB_PASSWORD" ] && missing+=("DB_PASSWORD")
    [ -z "$LDAP_HOST" ] && missing+=("LDAP_HOST")
    [ -z "$LDAP_BIND_DN" ] && missing+=("LDAP_BIND_DN")
    [ -z "$LDAP_BIND_PW" ] && missing+=("LDAP_BIND_PW")
    [ -z "$LDAP_BASE_DN" ] && missing+=("LDAP_BASE_DN")
    [ -z "$JIRA_DOMAIN" ] && missing+=("JIRA_DOMAIN")
    [ -z "$JIRA_USER" ] && missing+=("JIRA_USER")
    [ -z "$JIRA_PASSWORD" ] && missing+=("JIRA_PASSWORD")
    [ -z "$SONAR_HOST" ] && missing+=("SONAR_HOST")
    [ -z "$SONAR_TOKEN" ] && missing+=("SONAR_TOKEN")
    [ -z "$GITHUB_TOOLS_APP_CLIENT_ID" ] && missing+=("GITHUB_TOOLS_APP_CLIENT_ID")
    [ -z "$GITHUB_TOOLS_APP_CLIENT_SECRET" ] && missing+=("GITHUB_TOOLS_APP_CLIENT_SECRET")
    [ -z "$GITHUB_WDF_APP_CLIENT_ID" ] && missing+=("GITHUB_WDF_APP_CLIENT_ID")
    [ -z "$GITHUB_WDF_APP_CLIENT_SECRET" ] && missing+=("GITHUB_WDF_APP_CLIENT_SECRET")
    [ -z "$OAUTH_ENCRYPTION_KEY" ] && missing+=("OAUTH_ENCRYPTION_KEY")
    [ -z "$JENKINS_P_USER" ] && missing+=("JENKINS_P_USER")
    [ -z "$JENKINS_ATOM_JAAS_TOKEN" ] && missing+=("JENKINS_ATOM_JAAS_TOKEN")
    [ -z "$JENKINS_GKECFSMULTICIS2_JAAS_TOKEN" ] && missing+=("JENKINS_GKECFSMULTICIS2_JAAS_TOKEN")
    [ -z "$JENKINS_ATOMPERF_JAAS_TOKEN" ] && missing+=("JENKINS_ATOMPERF_JAAS_TOKEN")

    echo "${missing[@]}"
}

# Validate required environment variables
echo "🔍 Validating required environment variables..."
MISSING_VARS=($(check_required_vars))

# If any variables are still missing, exit with error
if [ ${#MISSING_VARS[@]} -ne 0 ]; then
    echo ""
    echo "❌ ERROR: Missing required environment variables:"
    echo ""
    for var in "${MISSING_VARS[@]}"; do
        echo "   • $var"
    done
    echo ""

    if [ -f "$ENV_PATH" ]; then
        echo "💡 Your .env file is missing these variables. Add them:"
        echo ""
        for var in "${MISSING_VARS[@]}"; do
            echo "export $var=\"your-value-here\""
        done
    else
        echo "💡 Create a .env file with the missing variables:"
        echo ""
        echo "cat > $ENV_PATH << 'EOF'"

        # Show environment-specific secrets
        if [ "$DEPLOY_ENVIRONMENT" == "dev" ]; then
            echo "# DEV Environment Secrets"
            echo "export JWT_SECRET_DEV=\$(openssl rand -base64 32)"
            echo "export OAUTH_ENCRYPTION_KEY_DEV=\$(openssl rand -hex 32)"
            echo "export GITHUB_TOOLS_APP_CLIENT_ID_DEV=\"your-github-tools-dev-id\""
            echo "export GITHUB_TOOLS_APP_CLIENT_SECRET_DEV=\"your-github-tools-dev-secret\""
            echo "export GITHUB_WDF_APP_CLIENT_ID_DEV=\"your-github-wdf-dev-id\""
            echo "export GITHUB_WDF_APP_CLIENT_SECRET_DEV=\"your-github-wdf-dev-secret\""
        else
            echo "# PROD Environment Secrets"
            echo "export JWT_SECRET_PROD=\$(openssl rand -base64 32)"
            echo "export OAUTH_ENCRYPTION_KEY_PROD=\$(openssl rand -hex 32)"
            echo "export GITHUB_TOOLS_APP_CLIENT_ID_PROD=\"your-github-tools-prod-id\""
            echo "export GITHUB_TOOLS_APP_CLIENT_SECRET_PROD=\"your-github-tools-prod-secret\""
            echo "export GITHUB_WDF_APP_CLIENT_ID_PROD=\"your-github-wdf-prod-id\""
            echo "export GITHUB_WDF_APP_CLIENT_SECRET_PROD=\"your-github-wdf-prod-secret\""
        fi
        echo ""
        echo "# Shared Services (same for all environments)"
        echo "export DB_PASSWORD=\$(openssl rand -base64 32)"
        echo "export LDAP_HOST=\"ldap.example.com\""
        echo "export LDAP_BIND_DN=\"CN=ServiceAccount,OU=Users,DC=example,DC=com\""
        echo "export LDAP_BIND_PW=\"your-ldap-password\""
        echo "export LDAP_BASE_DN=\"DC=example,DC=com\""
        echo "export JIRA_DOMAIN=\"jira.example.com\""
        echo "export JIRA_USER=\"service-account@example.com\""
        echo "export JIRA_PASSWORD=\"your-jira-api-token\""
        echo "export SONAR_HOST=\"https://sonarqube.example.com\""
        echo "export SONAR_TOKEN=\"your-sonar-token\""
        echo "export JENKINS_P_USER=\"your-jenkins-user\""
        echo "export JENKINS_ATOM_JAAS_TOKEN=\"your-jenkins-atom-token\""
        echo "export JENKINS_GKECFSMULTICIS2_JAAS_TOKEN=\"your-jenkins-gkecfsmulticis2-token\""
        echo "export JENKINS_ATOMPERF_JAAS_TOKEN=\"your-jenkins-atomperf-token\""
        echo ""
        echo "# Optional: AI Core Credentials (JSON array)"
        echo "# All fields are required: team, clientId, clientSecret, oauthUrl, apiUrl, resourceGroup"
        echo "export AI_CORE_CREDENTIALS='[{\"team\":\"team-name\",\"clientId\":\"client-id\",\"clientSecret\":\"secret\",\"oauthUrl\":\"https://tenant.authentication.sap.hana.ondemand.com/oauth/token\",\"apiUrl\":\"https://api.ai.prod.eu-central-1.aws.ml.hana.ondemand.com\",\"resourceGroup\":\"default\"}]'"
        echo "EOF"
    fi
    echo ""
    echo "Then run: ./deploy.sh $DEPLOY_ENVIRONMENT"
    echo ""
    exit 1
fi

echo "✅ All required environment variables are set"
echo ""

# Build --set flags from environment variables (all required vars are validated above)
echo "📝 Building deployment configuration..."
SET_FLAGS=""
SET_FLAGS="$SET_FLAGS --set config.deployEnvironment=$DEPLOY_ENVIRONMENT"
SET_FLAGS="$SET_FLAGS --set ingress.host=$INGRESS_HOST"
SET_FLAGS="$SET_FLAGS --set auth.redirectUrl=$AUTH_REDIRECT_URL"
SET_FLAGS="$SET_FLAGS --set jwt.secret=$JWT_SECRET"
SET_FLAGS="$SET_FLAGS --set token.secret=$TOKEN_SECRET"
SET_FLAGS="$SET_FLAGS --set postgresql.auth.password=$DB_PASSWORD"
SET_FLAGS="$SET_FLAGS --set ldap.host=$LDAP_HOST"
SET_FLAGS="$SET_FLAGS --set ldap.bindDN=$LDAP_BIND_DN"
SET_FLAGS="$SET_FLAGS --set ldap.bindPassword=$LDAP_BIND_PW"
SET_FLAGS="$SET_FLAGS --set ldap.baseDN=$LDAP_BASE_DN"
SET_FLAGS="$SET_FLAGS --set jira.domain=$JIRA_DOMAIN"
SET_FLAGS="$SET_FLAGS --set jira.user=$JIRA_USER"
SET_FLAGS="$SET_FLAGS --set jira.password=$JIRA_PASSWORD"
SET_FLAGS="$SET_FLAGS --set sonar.host=$SONAR_HOST"
SET_FLAGS="$SET_FLAGS --set sonar.token=$SONAR_TOKEN"
SET_FLAGS="$SET_FLAGS --set github.tools.clientId=$GITHUB_TOOLS_APP_CLIENT_ID"
SET_FLAGS="$SET_FLAGS --set github.tools.clientSecret=$GITHUB_TOOLS_APP_CLIENT_SECRET"
SET_FLAGS="$SET_FLAGS --set github.wdf.clientId=$GITHUB_WDF_APP_CLIENT_ID"
SET_FLAGS="$SET_FLAGS --set github.wdf.clientSecret=$GITHUB_WDF_APP_CLIENT_SECRET"
SET_FLAGS="$SET_FLAGS --set oauth.encryptionKey=$OAUTH_ENCRYPTION_KEY"
SET_FLAGS="$SET_FLAGS --set jenkins.pUser=$JENKINS_P_USER"
SET_FLAGS="$SET_FLAGS --set jenkins.atomJaasToken=$JENKINS_ATOM_JAAS_TOKEN"
SET_FLAGS="$SET_FLAGS --set jenkins.gkecfsmulticis2JaasToken=$JENKINS_GKECFSMULTICIS2_JAAS_TOKEN"
SET_FLAGS="$SET_FLAGS --set jenkins.atomperfJaasToken=$JENKINS_ATOMPERF_JAAS_TOKEN"

# Add AI Core credentials if present (optional)
# Write to temporary values file to avoid shell escaping issues
AI_CORE_VALUES_FILE=""
if [ ! -z "$AI_CORE_CREDENTIALS" ]; then
    echo "📦 AI Core credentials found, adding to deployment..."
    AI_CORE_VALUES_FILE=$(mktemp)
    cat > "$AI_CORE_VALUES_FILE" << EOF
aiCore:
  credentials: $AI_CORE_CREDENTIALS
EOF
fi

echo "✅ Configuration ready"
echo ""

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl."
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    echo "❌ Helm not found. Please install Helm 3."
    exit 1
fi

# Create namespace if it doesn't exist
echo "📦 Checking namespace..."
if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
    echo "   Creating namespace: $NAMESPACE"
    kubectl create namespace "$NAMESPACE"
else
    echo "   Namespace $NAMESPACE already exists"
fi

# Update Helm dependencies
echo ""
echo "📥 Updating Helm dependencies..."
helm dependency update "$CHART_DIR"

# Lint the chart
echo ""
echo "🔍 Linting Helm chart..."
helm lint "$CHART_DIR" -f "$VALUES_FILE"

# Check if release exists
if helm list -n "$NAMESPACE" | grep -q "$RELEASE_NAME"; then
    echo ""
    echo "📝 Release $RELEASE_NAME exists. Performing upgrade..."

    # Show diff if helm-diff plugin is installed
    if helm plugin list | grep -q "diff"; then
        echo ""
        echo "📊 Changes to be applied:"
        if [ ! -z "$AI_CORE_VALUES_FILE" ]; then
            helm diff upgrade "$RELEASE_NAME" "$CHART_DIR" \
                -f "$VALUES_FILE" \
                -f "$AI_CORE_VALUES_FILE" \
                $SET_FLAGS \
                $EXTRA_ARGS \
                -n "$NAMESPACE" || true
        else
            helm diff upgrade "$RELEASE_NAME" "$CHART_DIR" \
                -f "$VALUES_FILE" \
                $SET_FLAGS \
                $EXTRA_ARGS \
                -n "$NAMESPACE" || true
        fi
    fi

    if [ ! -z "$AI_CORE_VALUES_FILE" ]; then
        helm upgrade "$RELEASE_NAME" "$CHART_DIR" \
            -f "$VALUES_FILE" \
            -f "$AI_CORE_VALUES_FILE" \
            $SET_FLAGS \
            $EXTRA_ARGS \
            -n "$NAMESPACE" \
            --wait \
            --timeout 10m
    else
        helm upgrade "$RELEASE_NAME" "$CHART_DIR" \
            -f "$VALUES_FILE" \
            $SET_FLAGS \
            $EXTRA_ARGS \
            -n "$NAMESPACE" \
            --wait \
            --timeout 10m
    fi

    echo ""
    echo "✅ Upgrade completed successfully!"
else
    echo ""
    echo "📝 Installing new release: $RELEASE_NAME"

    # Dry run first
    echo "   Running dry-run validation..."
    helm install "$RELEASE_NAME" "$CHART_DIR" \
        -f "$VALUES_FILE" \
        $EXTRA_ARGS \
        -n "$NAMESPACE" \
        --dry-run > /dev/null

    read -p "Continue with installation? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [ ! -z "$AI_CORE_VALUES_FILE" ]; then
            helm install "$RELEASE_NAME" "$CHART_DIR" \
                -f "$VALUES_FILE" \
                -f "$AI_CORE_VALUES_FILE" \
                $SET_FLAGS \
                $EXTRA_ARGS \
                -n "$NAMESPACE" \
                --wait \
                --timeout 10m
        else
            helm install "$RELEASE_NAME" "$CHART_DIR" \
                -f "$VALUES_FILE" \
                $SET_FLAGS \
                $EXTRA_ARGS \
                -n "$NAMESPACE" \
                --wait \
                --timeout 10m
        fi

        echo ""
        echo "✅ Installation completed successfully!"
    else
        echo "❌ Installation cancelled."
        exit 0
    fi
fi

# Show status
echo ""
echo "📊 Release Status:"
helm status "$RELEASE_NAME" -n "$NAMESPACE"

# Show pods
echo ""
echo "📦 Pods:"
kubectl get pods -n "$NAMESPACE" -l "app.kubernetes.io/name=developer-portal-backend"

# Show service
echo ""
echo "🌐 Service:"
kubectl get svc -n "$NAMESPACE" -l "app.kubernetes.io/name=developer-portal-backend"

# Show ingress if enabled
if kubectl get ingress -n "$NAMESPACE" "$RELEASE_NAME" &> /dev/null; then
    echo ""
    echo "🌍 Ingress:"
    kubectl get ingress -n "$NAMESPACE" "$RELEASE_NAME"
fi

echo ""
echo "🎉 Deployment complete!"
echo ""
echo "Useful commands:"
echo "  View logs:       kubectl logs -f deployment/$RELEASE_NAME -n $NAMESPACE"
echo "  Port forward:    kubectl port-forward svc/$RELEASE_NAME 7008:7008 -n $NAMESPACE"
echo "  Helm status:     helm status $RELEASE_NAME -n $NAMESPACE"
echo "  Helm history:    helm history $RELEASE_NAME -n $NAMESPACE"
echo "  Uninstall:       helm uninstall $RELEASE_NAME -n $NAMESPACE"
echo ""

# Cleanup temporary files
if [ ! -z "$AI_CORE_VALUES_FILE" ] && [ -f "$AI_CORE_VALUES_FILE" ]; then
    rm -f "$AI_CORE_VALUES_FILE"
fi
