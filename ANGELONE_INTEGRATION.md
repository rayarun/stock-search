# Angel One SmartAPI Integration

This project includes support for Angel One SmartAPI as an alternate data provider for stock chart data.

## Credential Management

The Angel One integration uses a flexible credential provider system that supports:

### 1. Environment Variables (Default)
```bash
export ANGELONE_API_KEY="your_api_key"
export ANGELONE_CLIENT_CODE="your_client_code"
export ANGELONE_PASSWORD="your_password"
```

### 2. KMS Integration
Implement the `credentials.Provider` interface to integrate with your KMS solution:

```go
import "stock-search/credentials"

// Example: AWS KMS Provider
type AWSKMSProvider struct {
    kmsClient *kms.Client
}

func (p *AWSKMSProvider) GetCredential(key string) (string, error) {
    // Fetch encrypted credential from AWS Secrets Manager
    // Decrypt using KMS
    // Return decrypted value
}

// Use in handler
credProvider := &AWSKMSProvider{kmsClient: myKMSClient}
stockData, err := FetchAngelOneData(symbol, exchange, period, credProvider)
```

### 3. HashiCorp Vault Example
```go
type VaultProvider struct {
    vaultClient *vault.Client
    secretPath  string
}

func (p *VaultProvider) GetCredential(key string) (string, error) {
    secret, err := p.vaultClient.Logical().Read(p.secretPath + "/" + key)
    if err != nil {
        return "", err
    }
    return secret.Data[key].(string), nil
}
```

## Usage

### Using Yahoo Finance (Default)
```
GET /api/stock?symbol=RELIANCE&period=1D
```

### Using Angel One SmartAPI
```
GET /api/stock?symbol=RELIANCE&period=1D&provider=angelone&exchange=NSE
```

The system will automatically fall back to Yahoo Finance if Angel One fails.

## Symbol Token Mapping

Angel One uses numeric tokens instead of stock symbols. The current implementation includes a basic mapping for common stocks. For production use:

1. Fetch the master contract file from Angel One
2. Build a complete symbol-to-token mapping
3. Store in a database or cache

Example master contract API:
```
GET https://margincalculator.angelbroking.com/OpenAPI_File/files/OpenAPIScripMaster.json
```

## API Limits

Angel One has rate limits. The implementation includes:
- Token caching (JWT valid for 10 minutes)
- Automatic fallback to Yahoo Finance
- Error handling for rate limit responses

## Testing

Test with environment variables:
```bash
export ANGELONE_API_KEY="test_key"
export ANGELONE_CLIENT_CODE="test_code"
export ANGELONE_PASSWORD="test_password"

# Test the endpoint
curl "http://localhost:8080/api/stock?symbol=RELIANCE&period=1D&provider=angelone&exchange=NSE"
```

## Security Best Practices

1. **Never commit credentials** to version control
2. **Use KMS** for production deployments
3. **Rotate credentials** regularly
4. **Monitor API usage** to avoid rate limits
5. **Use HTTPS** for all API communications
