import * as SecureStore from 'expo-secure-store';
import { setupCredentials}  from "./auth";
import { clearCredentials } from "./storage";
import { deviceClient } from './api';
import { storeSettings } from './storage';
import firebase from '@react-native-firebase/app';

interface firebaseConfig {
  name: string;
  apiKey: string;
  appId: string;
  messagingSenderId: string;
  projectId: string;
  storageBucket: string;
  databaseURL: string | null;
}

export interface EnrollmentData {
  deviceId: string;
  enrollmentKey: string;
  instanceUrl: string;
  audience: string;
  firebaseConfig: firebaseConfig;
}


/**
 * Checks if the given data is an EnrollmentData object.
 *
 * @param data - The data to check
 * @returns True if the data is an EnrollmentData object, false otherwise
 */
export function isEnrollmentData(data: any): data is EnrollmentData {
  return (
    typeof data === 'object' &&
    typeof data.deviceId === 'string' &&
    typeof data.enrollmentKey === 'string' &&
    typeof data.instanceUrl === 'string' &&
    typeof data.audience === 'string' &&
    typeof data.firebaseConfig === 'object'
  );
}

/**
 * Enrolls the device with the given data.
 *
 * @param data - The enrollment data
 * @returns True if the device was enrolled, false otherwise
 */
export async function enroll(data: EnrollmentData): Promise<boolean> {
  const publicKey = await setupCredentials(data.deviceId, data.instanceUrl, data.audience);

  try {
    // Enroll the device.
    const res = await deviceClient(data.instanceUrl).enroll({
      publicKey: publicKey,
      enrollmentKey: data.enrollmentKey,
    });

    if(!res.settings) {
      return false;
    }

    // Store the device settings in localStorage
    await storeSettings(res.settings);

    // Can have clientId and databaseURL as well...
    /*firebase.initializeApp({
      apiKey: data.firebaseConfig.apiKey,
      appId: data.firebaseConfig.appId,
      messagingSenderId: data.firebaseConfig.messagingSenderId,
      projectId: data.firebaseConfig.projectId,
      storageBucket: data.firebaseConfig.storageBucket,
      databaseURL: data.firebaseConfig.databaseURL ?? '',
    }, { name: "INSTANCE_FCM" });*/
  } catch (e) {
    console.log('Failed to enroll device', e);
    await clearCredentials();
    return false;
  }
  return true;
}
