import {Component, useEffect, useState} from "react";
import {AuthContext, decrypt, getAuthContext, hasCredentials} from "../lib/auth";
import {enroll, EnrollmentData} from "../lib/enrollment";
import {ActivityIndicator, View, StyleSheet, Text} from "react-native";
import Enroll from "./Enroll";
import Call from "./Call";
import {deviceClient} from "../lib/api";
import messaging from "@react-native-firebase/messaging";

const authContextLifetime = 60 * 1000 // 60 seconds

export default function HomeCall(props: {

}){
  // Enrolled status
  let [enrolled, setEnrolled] = useState<boolean | null>(null);
  useEffect(() => {
    (async () => {
      setEnrolled(await hasCredentials());
    })();
  }, []);

  const [authContext, setAuthContext] = useState<AuthContext | null>(null)

  const renewAuthContext = async () => {
    setAuthContext(await getAuthContext());
  }

  useEffect(() => {
    if (!enrolled) {
      // No authContext before we are enrolled
      setAuthContext(null)
      return;
    }

    // Immediately fetch an auth-context
    // Fire and forget
    renewAuthContext();

    // Periodically renew the auth context
    const interval = setInterval(renewAuthContext, authContextLifetime);
    return () => clearInterval(interval);
  }, [enrolled]);

  let [fcmSetupDone, setFcmSetupDone] = useState<boolean>(false)

  // Setup FCM messaging
  useEffect(() => {
    if (authContext === null) {
      return;
    }
    // Permission request is required on IOS
    messaging().requestPermission();

    const submitFcmToken = (fcmToken: string) => {
      deviceClient(authContext.instanceUrl).updateNotificationToken({ notificationToken: fcmToken }, {
        headers: {
          Authorization: `Bearer ${authContext.deviceToken}`,
        }
      }).then(() => {
        setFcmSetupDone(true)
      })
    }

    messaging()
      .getToken()
      .then(submitFcmToken)

    return messaging().onTokenRefresh(submitFcmToken)
  }, [authContext]);


  const attemptEnrollment = async (data: EnrollmentData) => {
    setEnrolled(null);
    setEnrolled(await enroll(data));
  }

  const [lastMessage, setLastMessage] = useState<[type: string, message: string, timestamp: number] | undefined>(undefined)

  useEffect(() => {
    if (!fcmSetupDone) {
      return
    }
    return messaging().onMessage((message) => {
      if (message.data === undefined) {
        message.data = {}
      }

      const encryptedContent = message.data.encryptedContent
      if (typeof encryptedContent === 'string' && encryptedContent !== '') {
        decrypt(encryptedContent).then((decrypted) => {
          if (message.data === undefined) {
            message.data = {}
          }
          message.data.encryptedContent = decrypted
          setLastMessage(['fcm', JSON.stringify(message), message.sentTime ?? 0])
        })
      } else {
        setLastMessage(['fcm', JSON.stringify(message),  Date.now()])
      }
    })
  }, [fcmSetupDone]);

  const loadingScreen = (text: string) => {
    return (
      <View style={styles.loading}>
        <ActivityIndicator size="large" color="#002594" />
        <Text style={styles.loadingText}>
          {text}
        </Text>
      </View>
    );
  }

  // If we don't know if the user is enrolled yet, don't show anything
  if (enrolled === null) {
    return loadingScreen("Letar efter låset...");
  }

  if (!enrolled) {
    return <Enroll
      onEnroll={attemptEnrollment}
    />;
  }

  if (authContext === null) {
    return loadingScreen("Hämtar nycklarna...");
  }

  return (
    <Call authContext={authContext} lastMessage={lastMessage} />
  );
}

const styles = StyleSheet.create({
  loading: {
    height: '100%',
    width: '100%',
    textAlign: 'center',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
  },
  loadingText: {
    fontSize: 24,
    marginTop: 16,
    color: '#002594',
  }
});
