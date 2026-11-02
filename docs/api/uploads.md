# baseUrl/api/v1/uploads

File uploads (images, PDFs). Upload first, then reference the returned `id` from the resource that uses it (e.g. a profile's `pictureUploadId`).

Auth and 500 errors are listed in [Conventions](./conventions.md) and not repeated here.

---

## Types

```typescript
type UploadPurpose = "profile_picture" | "document_attachment";

type UploadKind = "image" | "document";
```

| Purpose               | Accepts       | Max files / request | Preview                             | Original kept |
| --------------------- | ------------- | ------------------- | ----------------------------------- | ------------- |
| `profile_picture`     | images        | 1                   | 512×512 WebP, cropped to a square   | no            |
| `document_attachment` | images or PDF | 10                  | fits in 1024×1024 WebP, not cropped | yes           |

`profile_picture` is **preview-only**: once the preview is made, the original is discarded, so `originalUrl` is `null` and `mimeType` is `image/webp`. Previews keep transparency: a transparent PNG stays transparent in its WebP preview.

`document_attachment` has nothing to attach to in the starter yet — the app built on it adds the resource that references it; until then such uploads are deleted as pending after 24 hours.

Accepted formats (detected from file **content**, not extension or the part's Content-Type):

| Format | Kind       | Max size |
| ------ | ---------- | -------- |
| JPEG   | `image`    | 5 MiB    |
| PNG    | `image`    | 5 MiB    |
| WebP   | `image`    | 5 MiB    |
| PDF    | `document` | 5 MiB    |

Images must also be at most **10000 px** wide and tall.

---

## POST /

- ### Description:

  Upload one or more files for a purpose. Each file is stored with its original plus (for images) a WebP preview; preview-only purposes (`profile_picture`) keep just the preview.

- ### Constraints:

  - `Content-Type: multipart/form-data` (let the browser/`FormData` set the boundary — don't set the header by hand).
  - Whole request ≤ **55 MiB**.
  - Request may take up to 2 minutes server-side; don't use a short client timeout.
  - **Uploads are pending until attached.** An upload not referenced by a resource (e.g. profile `pictureUploadId`) within **24 hours** is deleted, files included.
  - **Attached uploads live as long as something uses them.** Once no resource references an upload any more (picture replaced or removed, account deleted), it is deleted, files included, within about **24 hours**.
  - An upload can only be attached by the user who made it, and only to a resource matching its purpose.

- ### Authorization:

  Bearer access token.

- ### Request:

  Multipart form fields:

```
purpose: UploadPurpose   // text field, required
files:   File            // file field, repeat for each file (same name "files")
```

Example:

```typescript
const form = new FormData();
form.append("purpose", "profile_picture");
form.append("files", pictureFile);

await fetch(`${baseUrl}/api/v1/uploads`, {
  method: "POST",
  headers: { Authorization: `Bearer ${accessToken}` },
  body: form,
});
```

- ### Response:

  `201 Created` — one entry per file, in the order sent.

```typescript
interface UploadFilesResponse {
  uploads: UploadedFile[];
}

interface UploadedFile {
  id: string; // uuid — reference this from the resource using the file
  purpose: UploadPurpose;
  kind: UploadKind;
  mimeType: string; // of the stored file: image/jpeg | image/png | image/webp | application/pdf; always image/webp for preview-only purposes
  originalUrl: string | null; // public URL of the stored original; null for preview-only purposes
  previewUrl: string | null; // WebP preview for images; null for PDFs
}
```

```json
{
  "uploads": [
    {
      "id": "9a8b7c6d-5e4f-4a3b-2c1d-0e9f8a7b6c5d",
      "purpose": "document_attachment",
      "kind": "image",
      "mimeType": "image/png",
      "originalUrl": "https://storage.googleapis.com/<bucket>/.../original.png",
      "previewUrl": "https://storage.googleapis.com/<bucket>/.../preview.webp"
    },
    {
      "id": "1f2e3d4c-5b6a-4978-8695-a4b3c2d1e0f9",
      "purpose": "document_attachment",
      "kind": "image",
      "mimeType": "image/jpeg",
      "originalUrl": "https://storage.googleapis.com/<bucket>/.../original.jpg",
      "previewUrl": "https://storage.googleapis.com/<bucket>/.../preview.webp"
    }
  ]
}
```

Prefer `previewUrl` for thumbnails/display; fall back to `originalUrl` when it's `null`. Preview-only uploads (`profile_picture`) answer `originalUrl: null` and `mimeType: "image/webp"`.

- ### Errors:

  Validation is all-or-nothing: every file is checked before any is stored, so one bad file rejects the whole request with nothing saved. (A `500` mid-storage may leave earlier files stored as pending — they're swept after 24h; just retry.)

| Status | Code                                | When                                                                                        |
| ------ | ----------------------------------- | ------------------------------------------------------------------------------------------- |
| 400    | `request_body_invalid`              | Multipart body malformed or cut short                                                       |
| 413    | `request_body_too_large`            | Whole request over 55 MiB                                                                   |
| 415    | `unsupported_content_type`          | Not `multipart/form-data`                                                                   |
| 400    | `upload_purpose_invalid`            | `purpose` missing or not one of the known values                                            |
| 400    | `upload_files_missing`              | No `files` part                                                                             |
| 400    | `upload_too_many_files`             | More files than the purpose allows                                                          |
| 400    | `upload_file_type_unsupported`      | Not JPEG/PNG/WebP/PDF, or a kind the purpose doesn't accept (e.g. PDF as `profile_picture`) |
| 413    | `upload_file_too_large`             | A file over 5 MiB                                                                           |
| 400    | `upload_image_dimensions_too_large` | Image wider or taller than 10000 px                                                         |
| 400    | `upload_file_unreadable`            | Image content corrupt / undecodable                                                         |

Messages name the offending file, handy for per-file UI feedback:

```json
{
  "code": "upload_file_too_large",
  "message": "file \"scan.png\" is 7340032 bytes: expected a file no larger than the limit for its type of 5242880 bytes",
  "details": null
}
```

```json
{
  "code": "upload_file_type_unsupported",
  "message": "file \"contract.pdf\" is a document, not accepted as profile_picture: expected a file kind the upload purpose accepts",
  "details": null
}
```

```json
{
  "code": "upload_too_many_files",
  "message": "2 files: expected no more files than the upload purpose accepts per request of 1",
  "details": null
}
```
