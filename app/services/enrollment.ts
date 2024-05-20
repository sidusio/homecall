import {clearCredentials, setupCredentials} from "./auth";


interface EnrollmentData {
  deviceId: string;
  enrollmentKey: string;
  instanceUrl: string;
  audience: string;
}

function isEnrollmentData(data: any): data is EnrollmentData {
  return (
    typeof data === 'object' &&
    typeof data.deviceId === 'string' &&
    typeof data.enrollmentKey === 'string' &&
    typeof data.instanceUrl === 'string' &&
    typeof data.audience === 'string'
  );
}

export {
  enroll,
  EnrollmentData,
  isEnrollmentData
}


/**
 * Enrolls the device with the given data
 */
async function enroll(data: EnrollmentData): Promise<boolean> {
  const publicKey = await setupCredentials(data.deviceId, data.instanceUrl, data.audience);

  try {
    // TODO: call the enrollment API with the public key
    console.log('Generated keys', publicKey);
    // todo: store device settings
  } catch (e) {
    await clearCredentials();
    return false;
  }
  return true;
}
