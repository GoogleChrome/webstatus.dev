/**
 * Copyright 2023 Google LLC
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

import createClient, { type FetchOptions } from 'openapi-fetch'
import { type components, type paths } from 'webstatus.dev-backend'

// TODO. Remove once not behind UbP
const temporaryFetchOptions: FetchOptions<unknown> = {
  credentials: 'include'
}

export class APIClient {
  private readonly client: ReturnType<typeof createClient<paths>>
  constructor(baseUrl: string) {
    this.client = createClient<paths>({ baseUrl })
  }

  public async getFeature(
    featureId: string
  ): Promise<components['schemas']['Feature']> {
    const { data, error } = await this.client.GET('/v1/features/{feature_id}', {
      ...temporaryFetchOptions,
      params: { path: { feature_id: featureId } }
    })
    if (error != null) {
      throw new Error(error.message)
    }
    return data
  }

  public async getFeatures(): Promise<
    components['schemas']['FeaturePage']['data']
  > {
    const { data, error } = await this.client.GET('/v1/features', {
      params: {},
      ...temporaryFetchOptions
    })
    if (error != null) {
      throw new Error(error.message)
    }
    return data.data
  }
}
