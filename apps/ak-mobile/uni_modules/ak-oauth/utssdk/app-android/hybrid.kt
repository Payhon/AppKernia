package uts.sdk.modules.akOauth

import android.app.Activity
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import io.dcloud.uts.UTSAndroid

/** Android China build bridge. It deliberately has no Google SDK reference. */
object AkOAuthAndroidBridge {
    private const val META_VARIANT = "com.appkernia.oauth.variant"
    private const val CONSUMED = "com.appkernia.oauth.return.consumed"

    @JvmStatic fun buildVariant(): String {
        val activity = UTSAndroid.getUniActivity() as? Activity ?: return "android_china"
        val info = activity.packageManager.getApplicationInfo(activity.packageName, PackageManager.GET_META_DATA)
        return if (info.metaData?.getString(META_VARIANT) == "android_google") "android_google" else "android_china"
    }

    @JvmStatic fun openBrowser(url: String): Boolean {
        if (!url.startsWith("https://")) return false
        val activity = UTSAndroid.getUniActivity() as? Activity ?: return false
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url)).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        if (intent.resolveActivity(activity.packageManager) == null) return false
        return runCatching { activity.startActivity(intent); true }.getOrDefault(false)
    }

    @JvmStatic fun takeReturnURL(): String {
        val activity = UTSAndroid.getUniActivity() as? Activity ?: return ""
        val intent = activity.intent ?: return ""
        if (intent.getBooleanExtra(CONSUMED, false)) return ""
        val value = intent.dataString.orEmpty()
        if (value.isBlank()) return ""
        intent.putExtra(CONSUMED, true)
        return value
    }

    @JvmStatic fun authorizeGoogle(_serverClientId: String, _nonce: String, _success: (String) -> Unit, failure: (String) -> Unit) {
        failure("native_sdk_unavailable")
    }
}
