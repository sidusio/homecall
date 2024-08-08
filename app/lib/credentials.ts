import AsyncStorage from "@react-native-async-storage/async-storage";
import * as SecureStore from "expo-secure-store";

const appNamespace = 'io.sidus.homecall';
const credentialsSecureStoreTag = appNamespace + '.credentials';
const hasCredentialsStorageTag = appNamespace +'.hasCredentials';

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

export async function clearCredentials(): Promise<void> {
  await AsyncStorage.removeItem(hasCredentialsStorageTag);
  await SecureStore.deleteItemAsync(credentialsSecureStoreTag);
}

export async function storeCredentials (credentials: Credentials): Promise<void> {
  await SecureStore.setItemAsync(credentialsSecureStoreTag, JSON.stringify(credentials));
  // Store flag in AsyncStorage so that we can use it to clear the credentials if the app has been uninstalled
  await AsyncStorage.setItem(hasCredentialsStorageTag, 'true');
}

export async function getCredentials(): Promise<Credentials> {
  const hasCredentials = await AsyncStorage.getItem(hasCredentialsStorageTag);
  if (hasCredentials !== 'true') {
    // If the app has been uninstalled, the credentials should be cleared
    await clearCredentials();
  }

  const credentialsData = await SecureStore.getItemAsync(credentialsSecureStoreTag);
  if (!credentialsData) {
    throw new Error('No credentials found');
  }

  const credentials = JSON.parse(credentialsData);
  if (!isCredentials(credentials)) {
    await clearCredentials();
    throw new Error('Invalid credentials, clearing them');
  }

  return credentials;
}
