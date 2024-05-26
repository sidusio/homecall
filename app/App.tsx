import applyGlobalPolyfills from "./globals" // <-- Change the path
import { View } from 'react-native';
import Enroll from "./views/Enroll";
import { useKeepAwake } from 'expo-keep-awake';
import { useState, useEffect } from 'react';
import { getApiToken, hasCredentials } from "./services/auth";
import { EnrollmentData, enroll, getSettings } from "./services/enrollment";
import Call from "./views/Call";

applyGlobalPolyfills() // Hopefully make TextEncoder available

export default function App() {
  useKeepAwake(); // Keep the screen awake

  let [enrolled, setEnrolled] = useState<boolean | null>(null);
  const [apiToken, setApiToken] = useState<string>('');
  const [instanceUrl, setInstanceUrl] = useState<string>('');
  const [deviceId, setDeviceId] = useState<string>('');
  const [settings, setSettings] = useState<any>({});

  /**
   * Renews the API token.
   */
  const renewToken = async () => {
    const [token, url, deviceId] = await getApiToken();
    setApiToken(token);
    setInstanceUrl(url);
    setDeviceId(deviceId);
  }

  /**
   * Attempts to enroll the device.
   *
   * @param data - The enrollment data
   */
  const attemptEnrollment = async (data: EnrollmentData) => {
    setEnrolled(null);
    setEnrolled(await enroll(data));
  }

  // Check if the user is enrolled.
  useEffect(() => {
    (async () => {
      setEnrolled(await hasCredentials());
    })();
  }, []);

  // If the user is enrolled, renew the token and get the settings.
  useEffect(() => {
    if (!enrolled) {
      return;
    }

    renewToken();
    getSettings().then((settings) => {
      setSettings(settings);
    });

    const interval = setInterval(renewToken, 60 * 1000);
    return () => clearInterval(interval);
  }, [enrolled]);

  // View rendering.
  // If we don't know if the user is enrolled yet, don't show anything
  if (enrolled === null) {
    return <View />;
  }

  if (!enrolled) {
    return <Enroll
      onEnroll={attemptEnrollment}
    />;
  }

  if (!apiToken || !instanceUrl ) {
    return <View />;
  }

  return (
    <Call instanceUrl={instanceUrl} token={apiToken} deviceId={deviceId} settings={settings} />
  );
}
