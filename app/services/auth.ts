import * as SecureStore from 'expo-secure-store';
import { RSA } from 'react-native-rsa-native';
import base64url from "base64url";

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

export {
  setupCredentials,
  getApiToken,
  hasCredentials,
  clearCredentials,
}


/**
 * Sets up the credentials for the device
 * @returns PEM  encoded (SPKI) RSA public key for enrollment.
 */
async function setupCredentials(deviceId: string, instanceUrl: string, audience: string): Promise<string> {
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
async function getApiToken(): Promise<[string, string, string]> {
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

async function hasCredentials(): Promise<boolean> {
  try {
    await getCredentials();
    return true;
  } catch (error) {
    return false;
  }
}

async function clearCredentials(): Promise<void> {
  await SecureStore.deleteItemAsync(homecallSecureStoreTag);
}


const homecallSecureStoreTag = 'io.sidus.homecall.credentials';

async function storeCredentials (credentials: Credentials): Promise<void> {
  await SecureStore.setItemAsync(homecallSecureStoreTag, JSON.stringify(credentials));
}

async function getCredentials(): Promise<Credentials> {
  const credentialsData = await SecureStore.getItemAsync(homecallSecureStoreTag);

  if (!credentialsData) {
    throw new Error('No credentials found');
  }

  const credentials = JSON.parse(credentialsData);

  if (!isCredentials(credentials)) {
    clearCredentials();
    throw new Error('Invalid credentials, clearing them');
  }

  return credentials;
}
