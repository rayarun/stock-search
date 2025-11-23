# Stock Symbol Search Engine

A simple Go-based stock symbol search engine.

## Prerequisites

- Go 1.21 or higher

## Setup

1.  Initialize the module (if not already done):
    ```bash
    go mod tidy
    ```

## Running the Server

To start the server:

```bash
go run main.go
```

The server will start on port 8080.

## API Usage

### Search for a Stock

**Endpoint:** `GET /search`

**Query Parameters:**
- `q`: The search query (symbol or name prefix/substring)

**Example:**

```bash
curl "http://localhost:8080/search?q=Apple"
```

**Response:**

```json
[
  {
    "symbol": "AAPL",
    "name": "Apple Inc.",
    "exchange": "NASDAQ",
    "type": "Stock"
  }
]
```
