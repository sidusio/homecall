import {Platform, StyleSheet} from 'react-native';
import {WebView} from 'react-native-webview';
import {useEffect, useState} from 'react';
import Constants from "expo-constants";
import {AuthContext} from "../lib/auth";

interface WebViewRef {
  injectJavaScript(js: string): void
  reload(): void
}

const injectHomecallDeviceToken = async (token: string, webViewRef?: WebViewRef) => {
  if (!webViewRef) {
    return;
  }
  setTimeout(() => {
    webViewRef.injectJavaScript(`
      window.localStorage.setItem('homecallDeviceToken', '${token}');
    `);
  }, 100)

}

const injectHomecallAppData = async (deviceId: string, webViewRef?: WebViewRef) => {
  if (!webViewRef) {
    return;
  }

  const homecallAppData = JSON.stringify({
    deviceId: deviceId,
    appVersion: Constants.expoConfig?.version,
    platform: Platform.OS,
    platformVersion: Platform.Version,
    devMode: __DEV__,
    sessionId: Constants.sessionId,
  });

  webViewRef.injectJavaScript(`
    window.homecallAppData = ${homecallAppData};
  `);
}

const injectDebugging = (webViewRef?: WebViewRef) => {
  if (!webViewRef) {
    return;
  }

  const debugging = `
     // Debug
     console = new Object();
     console.log = function(log) {
      window.ReactNativeWebView.postMessage(JSON.stringify(log));
      return true;
     };
     console.debug = console.log;
     console.info = console.log;
     console.warn = console.log;
     console.error = console.log;
     `;

  webViewRef.injectJavaScript(debugging);
}

const onWebviewMessage = (event: { nativeEvent: { data: string; }; }) => {
  console.log('WebView: ' + event.nativeEvent.data);
}

export default function Call(props: {
  authContext: AuthContext,
  lastMessage?: [type: string, message: string, timestamp: number],
}){
  const { authContext, lastMessage } = props;

  const [webViewRef, setWebViewRef] = useState<WebViewRef | undefined>(undefined);

  // Pass over messages to webview
  useEffect(() => {
    if (lastMessage == undefined) {
      return
    }
    if (webViewRef == undefined) {
      return;
    }

    const [type, message, timestamp] = lastMessage
    if(timestamp < Date.now() - 1000 * 20) {
      return;
    }

    webViewRef.injectJavaScript(`
      window.dispatchEvent(new CustomEvent('${type}', { detail: '${message}' }));
    `);
  }, [lastMessage]);


  // Inject token whenever
  useEffect(() => {
    injectHomecallDeviceToken(authContext.deviceToken, webViewRef);
  }, [authContext, webViewRef]);


  const initialLoad = () => {
    injectHomecallDeviceToken(authContext.deviceToken, webViewRef);
    injectHomecallAppData(authContext.deviceId, webViewRef);
    injectDebugging(webViewRef);
  }
  useEffect(() => {
    initialLoad()
  }, [webViewRef]);

  // Refresh nightly
  const [lastRefresh, setLastRefresh] = useState(Date.now());

  const attemptRefresh = () => {
    if (!webViewRef) {
      return;
    }
    const now = new Date();

    // if refreshed in the last 6 hours, don't refresh again
    if (lastRefresh > (now.valueOf() - 6 * 60 * 60 * 1000)) {
      return;
    }

    // Only refresh between 1am and 4am
    if (now.getHours() < 1 || now.getHours() > 4) {
      return;
    }

    webViewRef.reload();
    initialLoad();
    setLastRefresh(now.valueOf());
  }
  useEffect(() => {
    const interval = setInterval(attemptRefresh, 30 * 60 * 1000);
    return () => clearInterval(interval);
  }, [lastRefresh]);

  return (
    <>
      <WebView
        style={styles.container}
        source={{
          uri: trimSuffix(authContext.instanceUrl, '/api') + `/device`,
          headers: {
            "Accept-Language": "sv",
          }
        }}
        // @ts-ignore
        ref={setWebViewRef}
        onLoad={initialLoad}
        onMessage={onWebviewMessage}
        mediaPlaybackRequiresUserAction={ false }
        allowsInlineMediaPlayback={ true }
        // Required to make jitsi iframe api work on iOS (Version/16.2 Safari/605.1.15 worked latest).
        applicationNameForUserAgent={"Version/16.2 Safari/605.1.15"}
        mediaCapturePermissionGrantType={"grant"}
      />
    </>
  )
}

function trimSuffix(input: string, suffix: string) {
  if (!input.endsWith(suffix)) {
    return input
  }
  return input.slice(0, -suffix.length)
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    justifyContent: 'center',
  },
  text: {
    alignItems: 'center',
    padding: 40,
    margin: 20,
    backgroundColor: '#DDDDDD',
    borderRadius: 10,
  },
  button: {
    position: 'absolute',
    zIndex: 1,
    bottom: 90,
    right: 10,
    padding: 20,
    backgroundColor: 'rgb(67, 107, 177)',
  }
});
