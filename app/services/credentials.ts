export interface Credentials {
  deviceId: string;
  privateKey: string;
  instanceUrl: string;
  audience: string;
}

export function isCredentials(data: any): data is Credentials {
  return (
    typeof data === 'object' &&
    typeof data.deviceId === 'string' &&
    typeof data.privateKey === 'string' &&
    typeof data.instanceUrl === 'string' &&
    typeof data.audience === 'string'
  );
}
