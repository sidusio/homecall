export interface Config {
    jitsi: {
        domain: string;
    };
    sentry: {
        dsn: string;
    };
    auth0: {
        domain: string;
        clientId: string;
        audience: string;
        redirectUri: string;
    };
    firebase: {
        apiKey: string;
        appId: string;
        messagingSenderId: string;
        projectId: string;
        storageBucket: string;
    };
}

function isConfig (data: any): data is Config {
    return (
        typeof data === 'object' &&
        typeof data.jitsi === 'object' &&
        typeof data.jitsi.domain === 'string' &&
        typeof data.sentry === 'object' &&
        typeof data.sentry.dsn === 'string' &&
        typeof data.auth0 === 'object' &&
        typeof data.auth0.domain === 'string' &&
        typeof data.auth0.clientId === 'string' &&
        typeof data.auth0.audience === 'string' &&
        typeof data.auth0.redirectUri === 'string' &&
        typeof data.firebase === 'object' &&
        typeof data.firebase.apiKey === 'string' &&
        typeof data.firebase.appId === 'string' &&
        typeof data.firebase.messagingSenderId === 'string' &&
        typeof data.firebase.projectId === 'string' &&
        typeof data.firebase.storageBucket === 'string'
    );
}

export function allOptionsDefined (config: Config): boolean {
    return (
        config.jitsi.domain !== '' &&
        config.sentry.dsn !== '' &&
        config.auth0.domain !== '' &&
        config.auth0.clientId !== '' &&
        config.auth0.audience !== '' &&
        config.auth0.redirectUri !== '' &&
        config.firebase.apiKey !== '' &&
        config.firebase.appId !== '' &&
        config.firebase.messagingSenderId !== '' &&
        config.firebase.projectId !== '' &&
        config.firebase.storageBucket !== ''
    );
}

export const getConfig = async (): Promise<Config> => {
    let config = await fetch('/config.json', {
        method: 'GET',
        headers: {
            'Content-Type': 'application/json',
            'Accept': 'application/json'
        }
    }).then(response => response.json())

    // If in production, get the config from the config.json file.
    if(import.meta.env.PROD) {
        if(isConfig(config)) {
            return config;
        } else {
            throw new Error('Invalid config');
        }
    }

    // Get all the config from the environment variables.
    return {
        jitsi: {
            domain: import.meta.env.VITE_APP_JITSI_DOMAIN
        },
        sentry: {
            dsn: import.meta.env.VITE_APP_SENTRY_DSN
        },
        auth0: {
            domain: import.meta.env.VITE_APP_AUTH0_DOMAIN,
            clientId: import.meta.env.VITE_APP_AUTH0_CLIENT_ID,
            audience: import.meta.env.VITE_APP_AUTH0_AUDIENCE,
            redirectUri: import.meta.env.VITE_APP_AUTH0_REDIRECT_URI
        },
        firebase: {
            apiKey: import.meta.env.VITE_APP_FIREBASE_API_KEY,
            appId: import.meta.env.VITE_APP_FIREBASE_APP_ID,
            messagingSenderId: import.meta.env.VITE_APP_FIREBASE_MESSAGING_SENDER_ID,
            projectId: import.meta.env.VITE_APP_FIREBASE_PROJECT_ID,
            storageBucket: import.meta.env.VITE_APP_FIREBASE_STORAGE_BUCKET,
        }
    }
}
