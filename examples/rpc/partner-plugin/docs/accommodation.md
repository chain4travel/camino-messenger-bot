# Accommodation Product List V1 & V2
Used to retrieve a list of all accommodation products of a supplier.
Modified after is optional, if not provided, all products will be returned. If provided, only products modified after the provided timestamp will be returned.


### Request message example

```
{
    "header": {
        "base_header": {
            "end_user_wallet_address": "nulla cupidatat adipisicing",
            "version": {
                "major": 1435022022,
                "minor": 749426816,
                "patch": 106982721
            }
        }
    },
    "modified_after": {
        "nanos": 1643852629,
        "seconds": "76332021931"
    }
}
```

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

**Required fields:**
- supplier_codes (array of supplier codes to filter by)

**Optional fields:**
- languages (array of language codes to be provided in the response)
- modified_after (timestamp to filter by, only products modified after the provided timestamp will be returned)


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
    "header": {
        "base_header": {
            "end_user_wallet_address": "mollit",
            "version": {
                "major": 1631864601,
                "minor": 1460812095,
                "patch": 1365431389
            }
        }
    },
    "supplier_codes": [
        {
            "supplier_code": "HTL567890",
            "supplier_number": 789
        },
        {
            "supplier_code": "HTL123456",
            "supplier_number": 847
        }
    ],
    "languages": [
        9,
        12
    ],
    "modified_after": {
        "nanos": 1643852629,
        "seconds": "76332021931"
    }
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
                    "supplier_code": "HTL123456",
                    "supplier_number": 847
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

