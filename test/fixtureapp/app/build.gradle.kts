plugins {
    id("com.android.application")
}

// The version code comes from CI: a live readback of the internal track
// plus one. The local default stays tiny so a local build can never outrun
// the store sequence by accident.
val fixtureVersionCode = (project.findProperty("fixtureVersionCode") as String?)?.toInt() ?: 1

android {
    namespace = "com.itdeveapps.stepsshare.fixture"
    compileSdk = 35

    defaultConfig {
        // Must match the Play fixture app exactly; see internal/livesmoke.
        applicationId = "com.itdeveapps.stepsshare"
        minSdk = 26
        targetSdk = 35
        versionCode = fixtureVersionCode
        versionName = "fixture-$fixtureVersionCode"
    }

    signingConfigs {
        create("upload") {
            storeFile = System.getenv("FIXTURE_KEYSTORE")?.let { file(it) }
            storePassword = System.getenv("FIXTURE_KEYSTORE_PASSWORD")
            keyAlias = System.getenv("FIXTURE_KEY_ALIAS")
            keyPassword = System.getenv("FIXTURE_KEY_PASSWORD")
        }
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            signingConfig = signingConfigs.getByName("upload")
        }
    }
}
