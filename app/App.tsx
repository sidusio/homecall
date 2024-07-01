import applyGlobalPolyfills from "./globals" // <-- Change the path
import { StyleSheet, Text, View } from 'react-native';
import Enroll from "./views/Enroll";
import { useKeepAwake } from 'expo-keep-awake';
import { useState, useEffect } from 'react';
import {getApiToken, hasCredentials, setupCredentials, clearCredentials} from "./services/auth";
import {EnrollmentData, enroll, getSettings} from "./services/enrollment";
import Call from "./views/Call";

// Hopefully make TextEncoder available
applyGlobalPolyfills()

export default function App() {
  useKeepAwake();
  let [enrolled, setEnrolled] = useState<boolean | null>(null);

  useEffect(() => {
    (async () => {
      setEnrolled(await hasCredentials());
    })();
  }, []);


  const [apiToken, setApiToken] = useState<string>('');
  const [instanceUrl, setInstanceUrl] = useState<string>('');
  const [deviceId, setDeviceId] = useState<string>('');
  const [settings, setSettings] = useState<any>({});

  const renewToken = async () => {
    const [token, url, deviceId] = await getApiToken();
    setApiToken(token);
    setInstanceUrl(url);
    setDeviceId(deviceId);
  }

  useEffect(() => {
    //clearCredentials();
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


  const attemptEnrollment = async (data: EnrollmentData) => {
    setEnrolled(null);
    setEnrolled(await enroll(data));
  }

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

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
    alignItems: 'center',
    justifyContent: 'center',
  },
});
