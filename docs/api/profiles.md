# baseUrl/api/v1/profiles

The signed-in user's personal details. One profile per user, always addressed implicitly by the access token — no id in the path.

Common decoding, auth and 500 errors are listed in [Conventions](./conventions.md) and not repeated here.

---

## POST /

- ### Description:

  Create the signed-in user's profile. Done once, typically right after signup.

- ### Constraints:

  - One profile per user — a second create answers `409 profile_exists` (use `PUT` instead).
  - Both names required; trimmed; must not be blank.
  - Each name: at most 50 characters (after trimming); Latin or Cyrillic letters only, words joined by a single space or a single dash `-`. No leading/trailing separator, no digits, apostrophes or symbols. `John Smith`, `Taylor-Joy`, `Žukauskas`, `Пётр` pass; `-Taylor`, `John  Smith`, `O'Brien` fail.
  - `pictureUploadId` optional: the id of your own `profile_picture` upload (see [Uploads](./uploads.md)). Absent, `null` or blank → no picture.
  - Email verification **not** required.

- ### Authorization:

  Bearer access token.

- ### Request:

```typescript
interface CreateProfileRequest {
  firstName: string; // required
  lastName: string; // required
  pictureUploadId?: string | null; // profile_picture upload id
}
```

Example:

```json
{
  "firstName": "Jane",
  "lastName": "Doe"
}
```

- ### Response:

  `201 Created`

```typescript
interface ProfileResponse {
  id: string;
  firstName: string;
  lastName: string;
  picture: {
    uploadId: string | null;
    previewUrl: string | null; // 512×512 cropped WebP
    originalUrl: string | null; // always null: profile pictures keep only their preview
  }; // every field null when the profile has no picture
}
```

```json
{
  "id": "0b8d4c2e-7f1a-4a3b-8c9d-2e3f4a5b6c7d",
  "firstName": "Jane",
  "lastName": "Doe",
  "picture": {
    "uploadId": "7d2e5f3c-6a91-4b22-9c4d-3f8b0e7a5d26",
    "previewUrl": "https://storage.example/uploads/.../preview.webp",
    "originalUrl": null
  }
}
```

- ### Errors:

| Status | Code                             | When                                                                               |
| ------ | -------------------------------- | ---------------------------------------------------------------------------------- |
| 400    | `first_name_missing`             | `firstName` empty/blank                                                            |
| 400    | `last_name_missing`              | `lastName` empty/blank                                                             |
| 400    | `first_name_invalid`             | `firstName` breaks the name rules or is over 50 characters                         |
| 400    | `last_name_invalid`              | `lastName` breaks the name rules or is over 50 characters                          |
| 400    | `profile_picture_upload_invalid` | `pictureUploadId` malformed, unknown, not yours, or not a `profile_picture` upload |
| 409    | `profile_exists`                 | User already has a profile                                                         |

```json
{
  "code": "first_name_missing",
  "message": "first name is missing: expected a non-empty first name",
  "details": null
}
```

---

## PUT /

- ### Description:

  Replace the names and picture on the signed-in user's profile. Full replace — send every field each time; omitting `pictureUploadId` (or sending `null`) removes the picture.

- ### Constraints:

  - Profile must exist (create it with `POST` first).
  - `pictureUploadId` optional: the id of your own `profile_picture` upload. A replaced picture is deleted within about 24 hours.
  - Both names required; trimmed; must not be blank.
  - Each name: at most 50 characters (after trimming); Latin or Cyrillic letters only, words joined by a single space or a single dash `-`. No leading/trailing separator, no digits, apostrophes or symbols. `John Smith`, `Taylor-Joy`, `Žukauskas`, `Пётр` pass; `-Taylor`, `John  Smith`, `O'Brien` fail.

- ### Authorization:

  Bearer access token.

- ### Request:

```typescript
interface UpdateProfileRequest {
  firstName: string; // required
  lastName: string; // required
  pictureUploadId?: string | null; // profile_picture upload id
}
```

Example:

```json
{
  "firstName": "Jane",
  "lastName": "Smith"
}
```

- ### Response:

  `200 OK` — same shape as `POST`.

```json
{
  "id": "0b8d4c2e-7f1a-4a3b-8c9d-2e3f4a5b6c7d",
  "firstName": "Jane",
  "lastName": "Smith",
  "picture": {
    "uploadId": null,
    "previewUrl": null,
    "originalUrl": null
  }
}
```

- ### Errors:

| Status | Code                             | When                                                                               |
| ------ | -------------------------------- | ---------------------------------------------------------------------------------- |
| 400    | `first_name_missing`             | `firstName` empty/blank                                                            |
| 400    | `last_name_missing`              | `lastName` empty/blank                                                             |
| 400    | `first_name_invalid`             | `firstName` breaks the name rules or is over 50 characters                         |
| 400    | `last_name_invalid`              | `lastName` breaks the name rules or is over 50 characters                          |
| 400    | `profile_picture_upload_invalid` | `pictureUploadId` malformed, unknown, not yours, or not a `profile_picture` upload |
| 404    | `profile_not_found`              | No profile yet                                                                     |

```json
{
  "code": "profile_not_found",
  "message": "profile not found for user 6f1c2a8e-3b4d-4e5f-9a0b-1c2d3e4f5a6b",
  "details": null
}
```
