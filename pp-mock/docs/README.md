## PP-mock ( Partner Plugin Mock)

### Running partner plugin (pp-mock) example

See [README.MD](../../README.MD)for running partner plugin example.

### Request and Response messages examples

Can be retrieved by running the e2e tests as described in [README.MD](../../tests/e2e/README.MD) with `--debug` flag and taking a look at the output.

# ProductListRequest

Used to retrieve a list of all products of a supplier.

# ProductInfoRequest

Used to retrieve detailed information about a specific product.

# SearchRequest

Used to search availability for products based on the provided criteria.

e.g. AccommodationSearch

Search criteria:

- location_geo_tree (country, region, city_or_resort)
- travel_period (start_date, end_date)
- supplier_codes (array of supplier codes to filter by)

A travel period is required to search for accommodations (with limits of start/end values of now() / now() + 60 days)
There are restrictions on the travel period. Searching accommodations for travel period outside of mentioned dates is considered invalid.

In SearchParametersGeneric, Currency is required to return the result prices in the provided currency.

### Mock data service available supplier codes for testing AccommodationSearch:

- HOTEL123456
- HOTEL789012
- HOTEL345678
- HOTEL901234
- HOTEL567890

### Mock data service available languages for testing AccommodationSearch:

- 9 (English)
- 12 (German)
- 15 (Italian)
- 11 (French)

---

### Mock data service available geo tree locations for the AccommodationSearch (country, region, city_or_resort):

- country (13, 9, 90, 82, 32)
- region (Balearic Islands, Graubünden, Antalya, Berlin, Hawaii)
- city_or_resort (Mallorca, Arosa, Alanya, Berlin, Maui)

# Validation

Used for validating the availability of the search query.

# Mint

Used for minting a buyable token.
