import applyGlobalPolyfills from "./globals" // <-- Change the path
import {StyleSheet, View} from 'react-native';
import { useKeepAwake } from 'expo-keep-awake';
import * as Sentry from '@sentry/react-native';
import {StatusBar} from 'react-native';
import HomeCall from "./components/HomeCall";
import {ComponentType} from "react";

// Hopefully make TextEncoder available
applyGlobalPolyfills()

let app: ComponentType = () => {

  useKeepAwake();
  return (
    <View style={styles.view}>
      <StatusBar hidden />
      <HomeCall />
    </View>
  );
}

if(!__DEV__) {
  Sentry.init({
    dsn: 'https://9639d3406a54364151d90077a1a2020b@o4507538136170496.ingest.de.sentry.io/4507538144755792',


    // uncomment the line below to enable Spotlight (https://spotlightjs.com)
    // enableSpotlight: __DEV__,
  });
  app = Sentry.wrap(app);
}

export default app;

const styles = StyleSheet.create({
  view: {
    height: '100%',
  },
});
