# Digital Signature Final Project

Backend API for a Digital Signature Information System, developed as a final (capstone) project based on internship work at Universitas Al Azhar Indonesia (UAI). The system supports document upload, self-signing, and cross-signing (requesting signatures from other users), with a simulated digital certificate flow.

## Tech Stack

- Golang
- PostgreSQL
- REST API
- JWT Authentication
- Postman

## Main Features

- Login and logout
- User profile
- Role-based access control
- Upload PDF document
- Create signature request
- Approve or reject signature request
- Track document status
- Upload signature and initial image
- Digital certificate simulation

## Folder Structure

```
├── code          # backend source code
├── database      # database schema, seed, and ERD
├── docs          # technical documentation
├── laporan       # final project report
├── postman       # API collection and testing
├── references    # internship documents and guidance notes
└── screenshot    # implementation and testing evidence
```

## Getting Started

### Prerequisites

- [Go](https://go.dev/dl/) 1.27+
- PostgreSQL (or a hosted instance, e.g. Railway)
- [Postman](https://www.postman.com/downloads/) (for testing the API)

### Setup

1. Clone the repository:
   ```bash
   git clone <repo-url>
   cd <repo-folder>/code
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Create a `.env` file with your database connection string and JWT secret.
4. Run the schema and seed files in `database/` against your PostgreSQL instance.
5. Start the server:
   ```bash
   go run main.go
   ```

### API Testing

Import the collection from the `postman/` folder into Postman to explore and test the available endpoints.

## Documentation

- Full technical documentation: `docs/`
- Final project report: `laporan/`
- Database schema and ERD: `database/`

## Author

Neyza Maylanie Santosa. 
CEP-CCIT FTUI, Full Stack Developer - Information Technology.

## License

© 2026 Neyza Maylanie Santosa. All rights reserved.
