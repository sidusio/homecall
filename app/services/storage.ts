import AsyncStorage from "@react-native-async-storage/async-storage";
import * as SecureStore from "expo-secure-store";
import { Credentials, isCredentials } from "./credentials";
import {DeviceSettings, isDeviceSettings} from "./deviceSettings";

const credentialsSecureStoreTag = 'io.sidus.homecall.credentials';
const hasCredentialsStorageTag = 'io.sidus.homecall.hasCredentials';

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

const deviceSettingsStoreTag = 'io.sidus.homecall.deviceSettings';

export async function storeSettings(settings: DeviceSettings) {
  await AsyncStorage.setItem(deviceSettingsStoreTag, JSON.stringify(settings));
}

export async function getSettings(): Promise<DeviceSettings | boolean> {
  const settingsData = await AsyncStorage.getItem(deviceSettingsStoreTag);
  if (!settingsData) {
    throw new Error('No settings found');
  }

  const settings = JSON.parse(settingsData);
  if (!isDeviceSettings(settings)) {
    throw new Error('Invalid settings');
  }

  return settings
}
