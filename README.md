# logtail

logtail is an HTTP handler that serves log file contents.

Essential production features such as rate limiting, authentication can be
added by wrapping this handler with middleware. This handler only deals with
the file access.

# API

## HEAD & GET

### Request parameters

#### _offset_

If positive, start at byte offset relative to beginning of file.

If negative, start at byte offset relative to end of file, or entire file if beginning of file reached.

#### _limit_

Limit output to number of bytes. Must be positive.

#### _suffix_

Logrotate file suffix. Only integers >= 1 are supported.

### Response headers

#### LogTail-File-Length: {length}

The current length of the file in bytes. Expect that the length of the file may
increase OR decrease. A rotated log file may decrease in size, for example.

## GET only

### Response

#### 200 OK

_Requested file contents_

#### 204 No Content

The given offset is at or beyond the end of the file.

#### 400 Bad Request

Invalid request parameters.

#### 404 Not Found

The file doesn't exist.

#### 500 Internal Server Error

There was an error reading the requested file.
