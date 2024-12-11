# Business Process (How to Operate)

## Customer Section
- New customers register by providing their personal information (NIK, full name, legal name, birth details, salary)
```bash
curl --location 'http://localhost:8080/api/v1/customers' \
--header 'Content-Type: application/json' \
--data '{
    "nik": "1234567890123455",
    "full_name": "Gojo Satoru",
    "legal_name": "Gojo Satoru",
    "birth_place": "Tokyo",
    "birth_date": "2001-01-01T00:00:00Z",
    "salary": 10000000
}'
```
- Customers must submit required documents (KTP and selfie) for verification
```bash
curl --location 'http://localhost:8080/api/v1/customers/2594d283-f808-453d-992f-1bdab338bbde/documents' \
--header 'Content-Type: application/json' \
--data '{
    "document_type": "ktp",
    "document_url": "https://cdn.imgchest.com/files/j7mmczrenk7.png"
}'
```

## Credit Limit Assignment
Each customer can be assigned different credit limits based on tenor periods (1, 2, 3, or 6 months) <br>
Ex : 
- Budi limit : 100k (1 month), 200k (2 months), 500k (3 months), 700k (6 months)
- Annisa limit : 1M (1 month), 1.2M (2 months), 1.5M (3 months), 2M (6 months) <br>

These limits represent how much credit the customer can use for purchases

## Asset Management
The system maintains a catalog of available assets in three categories
- (White Goods (electronics, appliances)
- Motorcycles
- Cars

## Transaction
Transaction Process : Customer make purchases through multiple channels -> System checks if customer has sufficient credit limit for chosen tenor 
-> Calculates admin fees and interest -> Creates installment payment schedule
-> Records contract details
-> Updates customer's used credit limit

## Credit Limit Management
- System tracks used and available credit for each customer
- A customer can have multiple active transactions as long as they stay within their credit limits
- Credit limits are managed separately for different tenor periods
