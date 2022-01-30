import got from 'got';
import { Logger } from 'winston';

function ApiAuthorization(client_id: string | undefined, client_secret: string | undefined, logger: Logger): ApiAuthorization {

    if (client_id === undefined || client_secret === undefined) {
        throw new Error('Cannot find client secret or id.\nProvide them with CLIENT_ID and CLIENT_SECRET environment variables');
    }

    const authorization_uri_base = 'battle.net/oauth/token';

    const token_store = new Map<RegionCode, AccessToken>();

    /*
    const clientAccessToken: AccessToken = {
        access_token: '',
        token_type: '',
        expires_in: 0,
        scope: '',
        fetched: Date.now(),
        checkExpired: function (): boolean {
            let expired = true;
            const current_time = Date.now();
            const expire_time = this.fetched + (this.expires_in * 1000);
            if (current_time < expire_time) {
                expired = false;
            }
            return expired;
        },
    };*/

    /**
     * Get an oath token from blizzard, or use one we already have.
     */
    async function getAuthorizationToken(region: RegionCode): Promise<AccessToken> {
        if (!token_store.has(region)) {
            {
                token_store.set(region,
                    {
                        access_token: '',
                        token_type: '',
                        expires_in: 0,
                        scope: '',
                        fetched: Date.now(),
                        checkExpired: function (): boolean {
                            let expired = true;
                            const current_time = Date.now();
                            const expire_time = this.fetched + (this.expires_in * 1000);
                            if (current_time < expire_time) {
                                expired = false;
                            }
                            return expired;
                        }
                    }
                );
            }
        }

        const clientAccessToken = token_store.get(region);

        if (clientAccessToken == undefined) {
            throw new Error(`Cannot find or create access token for region ${region}`);
        }

        if (clientAccessToken.checkExpired()) {
            logger.debug('Access token expired, fetching fresh.');
            try {
                const auth_response: { body: any } = await got(`https://${region}.${authorization_uri_base}`, {
                    responseType: 'json',
                    method: 'POST',
                    username: client_id,
                    password: client_secret,
                    headers: {
                        'Connection': 'keep-alive'
                    },
                    form: {
                        'grant_type': 'client_credentials',
                    }
                });
                clientAccessToken.access_token = auth_response.body.access_token;
                clientAccessToken.token_type = auth_response.body.token_type;
                clientAccessToken.expires_in = auth_response.body.expires_in;
                clientAccessToken.scope = auth_response.body.scope;
                clientAccessToken.fetched = Date.now();

                token_store.set(region, clientAccessToken);
            } catch (error) {
                logger.error("An error was encountered while retrieving an authorization token: " + error, error);
            }
        }
        return clientAccessToken;
    }

    return Object.freeze({
        getAuthorizationToken
    });
}

export { ApiAuthorization };
