# TODO: Move Missing API Adapters To SDK

## Goal

Move Flashduty API endpoint adapters out of `flashduty-cli` and into `github.com/flashcatcloud/flashduty-sdk`.

## Tasks

- [x] Confirm every CLI method that bypasses `flashduty-sdk`.
- [x] Add SDK typed inputs, outputs, and methods for incident lifecycle endpoints currently implemented by CLI raw HTTP code.
- [x] Add SDK typed inputs, outputs, and methods for incident war-room endpoints.
- [x] Add SDK datasource discovery support for `POST /datasource/im/war-room-enabled/list`.
- [x] Add focused SDK tests for request body encoding and response decoding.
- [x] Keep SDK API names stable and simple enough for CLI consumption.

## Candidate Endpoints

- `/incident/unack`
- `/incident/wake`
- `/incident/remove`
- `/incident/disable-merge`
- `/incident/comment`
- `/incident/responder/add`
- `/incident/war-room/create`
- `/incident/war-room/list`
- `/incident/war-room/detail`
- `/incident/war-room/delete`
- `/incident/war-room/add-member`
- `/incident/war-room/default-observers`
- `/datasource/im/war-room-enabled/list`
