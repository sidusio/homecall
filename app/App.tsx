import applyGlobalPolyfills from "./globals" // <-- Change the path
import { StyleSheet, Text, View } from 'react-native';
import Enroll from "./views/Enroll";
import { useKeepAwake } from 'expo-keep-awake';
import { useState, useEffect } from 'react';
import { getApiToken, hasCredentials } from "./services/auth";
import { EnrollmentData, enroll } from "./services/enrollment";
import { getSettings } from "./services/storage";
import Call from "./views/Call";
import * as Sentry from '@sentry/react-native';
import {StatusBar} from 'react-native';

Sentry.init({
  dsn: 'https://9639d3406a54364151d90077a1a2020b@o4507538136170496.ingest.de.sentry.io/4507538144755792',

  // uncomment the line below to enable Spotlight (https://spotlightjs.com)
  // enableSpotlight: __DEV__,
});

// Hopefully make TextEncoder available
applyGlobalPolyfills()

function App() {
  <StatusBar hidden />

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

export default Sentry.wrap(App);
