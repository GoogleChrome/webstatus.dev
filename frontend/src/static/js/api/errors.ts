/**
 * Copyright 2024 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

export function createAPIError(error: unknown): ApiError {
  let message = 'Unknown error';
  let code = 500; // Default to Internal Server Error

  if (
    error instanceof Object &&
    'message' in error &&
    typeof error.message === 'string' &&
    'code' in error &&
    typeof error.code === 'number'
  ) {
    message = error.message;
    code = error.code;
  } else if (error instanceof Error) {
    message = error.message;
  }

  switch (code) {
    case 400:
      return new BadRequestError(message);
    case 401:
      return new UnauthorizedError(message);
    case 403:
      return new ForbiddenError(message);
    case 404:
      return new NotFoundError(message);
    case 429:
      return new RateLimitExceededError(message);
    case 500:
      return new InternalServerError(message);
    default:
      return new UnknownError(message);
  }
}

export class ApiError extends Error {
  code: number;
  constructor(message: string, code: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
  }
}

export class BadRequestError extends ApiError {
  constructor(message: string) {
    super(message, 400);
    this.name = 'BadRequestError';
  }
}

export class UnauthorizedError extends ApiError {
  constructor(message: string) {
    super(message, 401);
    this.name = 'UnauthorizedError';
  }
}

export class ForbiddenError extends ApiError {
  constructor(message: string) {
    super(message, 403);
    this.name = 'ForbiddenError';
  }
}

export class RateLimitExceededError extends ApiError {
  constructor(message: string) {
    super(message, 429);
    this.name = 'RateLimitExceededError';
  }
}

export class InternalServerError extends ApiError {
  constructor(message: string) {
    super(message, 500);
    this.name = 'InternalServerError';
  }
}

export class NotFoundError extends ApiError {
  constructor(message: string) {
    super(message, 404);
    this.name = 'NotFoundError';
  }
}

export class UnknownError extends ApiError {
  constructor(message: string) {
    super(message, 0);
    this.name = 'UnknownError';
  }
}
