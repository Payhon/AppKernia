package uts.sdk.modules.akPermissions

import android.Manifest
import android.app.Activity
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.provider.Settings
import io.dcloud.uts.UTSAndroid

object AkPermissionsAndroidBridge {
    private const val PREFS = "ak.permissions.v1"
    private const val NOTIFICATION_REQUESTED = "notifications.requested"
    private const val REQUEST_CODE = 0x4A51
    private val main = Handler(Looper.getMainLooper())

    @JvmStatic fun notificationStatus(): String {
        val context = context() ?: return "unavailable"
        val manager = context.getSystemService(Context.NOTIFICATION_SERVICE) as? NotificationManager ?: return "unavailable"
        if (Build.VERSION.SDK_INT < 33) return if (manager.areNotificationsEnabled()) "authorized" else "denied"
        if (context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) {
            return if (manager.areNotificationsEnabled()) "authorized" else "denied"
        }
        return if (requested(context)) "denied" else "not_determined"
    }

    @JvmStatic fun notificationCanRequest(): Boolean {
        if (Build.VERSION.SDK_INT < 33) return false
        val activity = UTSAndroid.getUniActivity() as? Activity ?: return false
        if (activity.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) return false
        return !requested(activity) || activity.shouldShowRequestPermissionRationale(Manifest.permission.POST_NOTIFICATIONS)
    }

    @JvmStatic fun requestNotifications(callback: (String) -> Unit) {
        val activity = UTSAndroid.getUniActivity() as? Activity ?: run { callback("unavailable"); return }
        if (Build.VERSION.SDK_INT < 33 || activity.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) {
            callback(notificationStatus()); return
        }
        activity.getSharedPreferences(PREFS, Context.MODE_PRIVATE).edit().putBoolean(NOTIFICATION_REQUESTED, true).apply()
        activity.requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), REQUEST_CODE)
        poll(0, callback)
    }

    @JvmStatic fun openNotificationSettings(): Boolean {
        val activity = UTSAndroid.getUniActivity() as? Activity ?: return false
        return runCatching {
            val intent = Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, activity.packageName)
            activity.startActivity(intent); true
        }.getOrElse {
            runCatching {
                activity.startActivity(Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.parse("package:${activity.packageName}"))); true
            }.getOrDefault(false)
        }
    }

    private fun poll(attempt: Int, callback: (String) -> Unit) {
        val activity = UTSAndroid.getUniActivity() as? Activity ?: run { callback("unavailable"); return }
        val granted = activity.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED
        val rationale = activity.shouldShowRequestPermissionRationale(Manifest.permission.POST_NOTIFICATIONS)
        if (granted || rationale || attempt >= 60) { callback(notificationStatus()); return }
        main.postDelayed({ poll(attempt + 1, callback) }, 250)
    }

    private fun requested(context: Context): Boolean = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getBoolean(NOTIFICATION_REQUESTED, false)
    private fun context(): Context? = UTSAndroid.getAppContext() as? Context
}
