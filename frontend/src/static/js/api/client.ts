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
}