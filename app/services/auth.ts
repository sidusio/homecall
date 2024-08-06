import { RSA } from 'react-native-rsa-native';
import base64url from "base64url";
import {getCredentials, storeCredentials} from "./storage";

const Buffer = require("buffer").Buffer;


interface Credentials {
  deviceId: string;
  privateKey: string;
  instanceUrl: string;
  audience: string;
}

function isCredentials(data: any): data is Credentials {
  return (
    typeof data === 'object' &&
    typeof data.deviceId === 'string' &&
    typeof data.privateKey === 'string' &&
    typeof data.instanceUrl === 'string' &&
    typeof data.audience === 'string'
  );
}


/**
 * Sets up the credentials for the device
 * @returns PEM  encoded (SPKI) RSA public key for enrollment.
 */
export async function setupCredentials(deviceId: string, instanceUrl: string, audience: string): Promise<string> {
  const keypair = await RSA.generateKeys(2048);

  const credentials: Credentials = {
    deviceId,
    privateKey: keypair.private,
    instanceUrl: instanceUrl,
    audience,
  };

  await storeCredentials(credentials);

  return keypair.public;
}

/**
 * Gets the credentials for the device
 * @returns The API token and the api url
 */
export async function getApiToken(): Promise<[string, string, string]> {
  const { privateKey, deviceId, instanceUrl, audience } = await getCredentials();

  const jwtHeader = {
    alg: 'RS256',
    typ: 'JWT'
  };

  const jwtPayload = {
    iss: 'homecall-device',
    sub: deviceId,
    aud: audience,
    exp: Math.floor(Date.now() / 1000) + 60 * 60,
    iat: Math.floor(Date.now() / 1000),
  };

  const jwtHeaderBase64 = base64url.fromBase64(Buffer.from(JSON.stringify(jwtHeader)).toString('base64'));
  const jwtPayloadBase64 = base64url.fromBase64(Buffer.from(JSON.stringify(jwtPayload)).toString('base64'));

  const jwtHeaderPayload = `${jwtHeaderBase64}.${jwtPayloadBase64}`;

  const signature = await RSA.signWithAlgorithm(jwtHeaderPayload, privateKey, 'SHA256withRSA')
  const sanitizedSignature = signature.replace(/\n/g, ''); // RSA library adds newlines to the signature, which is invalid for JWT
  const urlEncodedSignature = base64url.fromBase64(sanitizedSignature);

  const jwt = `${jwtHeaderPayload}.${urlEncodedSignature}`;

  return [jwt, instanceUrl, deviceId];
}

export async function hasCredentials(): Promise<boolean> {
  try {
    await getCredentials();
    return true;
  } catch (error) {
    return false;
  }
}
