import createClient from "openapi-fetch";
import { paths } from "webstatus.dev-backend";
export class Client {
    private readonly client: ReturnType<typeof createClient<paths>>;
    constructor(baseUrl: string) {
        this.client = createClient<paths>({baseUrl: baseUrl})
    }
    public async getFeatures() {
        const { data, error } = await this.client.GET("/v1/features", {
            params: {}
        });
        if (error) {
            throw new Error(error.message);
        }
        return data.data;
    }
    public async getFeature(feature_id:string) {
        const { data, error } = await this.client.GET("/v1/features/{feature_id}", {
            params: {path: {feature_id: feature_id}}
        });
        if (error) {
            throw new Error(error.message);
        }
        return data;
    }
}