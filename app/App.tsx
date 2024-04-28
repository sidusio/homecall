import { StyleSheet, Text, View } from 'react-native';
import Enroll from "./views/Enroll";
import { useKeepAwake } from 'expo-keep-awake';
import { useState, useEffect } from 'react';
import {getApiToken, hasCredentials, setupCredentials} from "./services/auth";
import {EnrollmentData, enroll} from "./services/enrollment";
import Call from "./views/Call";


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

  const renewToken = async () => {
    const [token, url] = await getApiToken();
    setApiToken(token);
    setInstanceUrl(url);
  }

  useEffect(() => {
    if (!enrolled) {
      return;
    }

    renewToken();

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
    <Call instanceUrl={instanceUrl} token={apiToken} />
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
