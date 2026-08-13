package com.xbs.app

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.runtime.LaunchedEffect
import androidx.navigation.compose.rememberNavController
import com.xbs.app.data.api.AuthEventBus
import com.xbs.app.data.local.TokenStore
import com.xbs.app.navigation.AppNavHost
import com.xbs.app.navigation.Routes
import com.xbs.app.ui.theme.XbsTheme
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.runBlocking
import javax.inject.Inject

@AndroidEntryPoint
class MainActivity : ComponentActivity() {

    @Inject lateinit var tokenStore: TokenStore
    @Inject lateinit var authEventBus: AuthEventBus

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        // demo 级启动路由判断：有 token 直接进发现页
        val start = runBlocking {
            if (tokenStore.token().isNullOrBlank()) Routes.LOGIN else Routes.DISCOVER
        }
        setContent {
            XbsTheme {
                val navController = rememberNavController()
                LaunchedEffect(Unit) {
                    authEventBus.events.collect {
                        navController.navigate(Routes.LOGIN) { popUpTo(0) { inclusive = true } }
                    }
                }
                AppNavHost(navController = navController, startDestination = start)
            }
        }
    }
}
