# Accommodation Product List V1 & V2

Used to retrieve a list of all accommodation products of a supplier.

### Response message example

```
{
    "properties": [
        {
            "product_codes": [...],
            "airports": [...],
            "last_modified": {...},
            "supplier_code": {...},
            "name": "Sunset Beach Resort & Spa",
            "chain": "Marriott",
            "category_rating": "CATEGORY_RATING_4_5",
            "category_unit": "CATEGORY_UNIT_STARS",
            "contact_info": {...},
            "coordinates": {...},
            "status": "PRODUCT_STATUS_NEW"
        },
        ...
    ],
    "header": null
}
```

# Accommodation Product Info V1 & V2

Used to retrieve detailed information about a specific accommodation product.

**Required message body fields:**

- supplier_codes (array of supplier codes to filter by)

**Optional message body fields:**

- languages (array of language codes to be provided in the response)

### Mock data service available supplier codes for testing:

- HTL123456
- HTL789012
- HTL345678
- HTL901234
- HTL567890

### Mock data service available languages for testing:

- 9 (English)
- 12 (German)
- 15 (Italian)
- 11 (French)

---

### Request message example

```
{
    "supplier_codes": [
        {
            "supplier_code": "HTL567890"
        },
        {
            "supplier_code": "HTL123456"
        }
    ],
    "languages": [
        9,
        12
    ],
}
```

### Response message example

```
{
    "properties": [
        {
            "images": [
                ...
            ],
            "videos": [
                ...
            ],
            "classifications": [
                ...
            ],
            "localized_descriptions": [
                ...
            ],
            "localized_room_descriptions": [
                ...
            ],
            "rooms": [
                ...
            ],
            "property": {
                "product_codes": [
                    ...
                ],
                "airports": [
                    "PMI",
                    "BCN"
                ],
                "last_modified": {
                    "seconds": "1710489022",
                    "nanos": 0
                },
                "supplier_code": {
                    "supplier_code": "HTL123456"
                },
                "name": "Sunset Beach Resort & Spa",
                "chain": "Marriott",
                "category_rating": "CATEGORY_RATING_4_5",
                "category_unit": "CATEGORY_UNIT_STARS",
                "contact_info": {
                    "address": [
                        ...
                    ],
                    "phones": [
                        ...
                    ],
                    "emails": [
                        ...
                    ],
                    "links": [
                        ...
                    ]
                },
                "coordinates": {
                    "latitude": 39.5696,
                    "longitude": 2.6502
                },
                "status": "PRODUCT_STATUS_NEW"
            },
            "payment_type": "MERCHANT"
        },
        ...
    ],
    "header": null
}
```

# Accommodation Search V1 & V2

Used to search availability for accommodation products based on the provided criteria.

Search criteria:

- location_geo_tree (country, region, city_or_resort)
- travel_period (start_date, end_date)
- supplier_codes (array of supplier codes to filter by)

There are restrictions on the travel period. Searching accommodations for travel period outside of mentioned dates is considered invalid - meaning there are no bookings available. Valid search query that will return results for this partner plugin:

- start_date (year, month, day) 2025-06-01
- end_date (year, month, day) 2025-06-30

Available supplier codes for the accommodation search:
- HTL123456
- HTL789012
- HTL345678
- HTL901234
- HTL567890

Available geo tree locations for the accommodation search (country, region, city_or_resort):

- country (13, 9, 90, 82, 32)
- region (Balearic Islands, Graubünden, Antalya, Berlin, Hawaii)
- city_or_resort (Mallorca, Arosa, Alanya, Berlin, Maui)

## Request message example
```
{
    "queries": [
        {
            "query_id": 1,
            "search_parameters_accommodation": {
                "location_geo_tree": {
                    "country": 13,
                    "region": "Balearic Islands",
                    "city_or_resort": "Mallorca"
                }
            },
            "travel_period": {
                "start_date": {
                    "year": 2025,
                    "month": 6,
                    "day": 15
                },
                "end_date": {
                    "year": 2025,
                    "month": 6,
                    "day": 18
                }
            }
        },
        {
            "query_id": 2,
            "search_parameters_accommodation": {
                "supplier_codes": [
                    {
                        "supplier_code": "HTL123456"
                    }
                ]
            },
            "travel_period": {
                "start_date": {
                    "year": 2025,
                    "month": 6,
                    "day": 15
                },
                "end_date": {
                    "year": 2025,
                    "month": 6,
                    "day": 18
                }
            }
        }
    ],
    "search_parameters_generic": {
        "currency": {
            "native_token": {}
        }
    }
}
```

# Validation

Used for validating the availability of the search query (accommodation search).

## Request message example
```
{
    "validation_object": {
        "search_identifier": {
            "result_id": 1,
            "search_id": {
                "value": "a3e2fb81-b632-4533-bfa4-86912847b5c8"
            }
        }
    }
}
```

# Mint

## Request message example
```
{
    "buyer_address": "0x1d32f368d8dc947270576773cc5E4778D7cA30Ba",
    "validation_id": {
        "value": "d731c098-826d-422c-b964-02775427f604"
    }
}
```
