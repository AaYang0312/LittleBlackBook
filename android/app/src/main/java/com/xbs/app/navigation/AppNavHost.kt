package com.xbs.app.navigation

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Explore
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.Icon
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.navArgument
import com.xbs.app.ui.auth.LoginScreen
import com.xbs.app.ui.auth.RegisterScreen
import com.xbs.app.ui.detail.DetailScreen
import com.xbs.app.ui.discover.DiscoverScreen
import com.xbs.app.ui.following.FollowingScreen
import com.xbs.app.ui.profile.ProfileScreen
import com.xbs.app.ui.publish.PublishScreen

/** 临时占位页，后续 Task 逐个替换（搜索 TASK-REPLACE）。 */
@Composable
private fun StubScreen(name: String) {
    Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { Text(name) }
}

@Composable
fun AppNavHost(navController: NavHostController, startDestination: String) {
    val backStackEntry by navController.currentBackStackEntryAsState()
    val currentRoute = backStackEntry?.destination?.route

    Scaffold(
        bottomBar = {
            if (currentRoute in Routes.TAB_ROUTES) {
                NavigationBar {
                    NavigationBarItem(
                        selected = currentRoute == Routes.DISCOVER,
                        onClick = { navController.navigate(Routes.DISCOVER) { popUpTo(Routes.DISCOVER) { inclusive = true }; launchSingleTop = true } },
                        icon = { Icon(Icons.Default.Explore, contentDescription = "发现") },
                        label = { Text("发现") },
                    )
                    NavigationBarItem(
                        selected = currentRoute == Routes.FOLLOWING,
                        onClick = { navController.navigate(Routes.FOLLOWING) { launchSingleTop = true } },
                        icon = { Icon(Icons.Default.Favorite, contentDescription = "关注") },
                        label = { Text("关注") },
                    )
                    NavigationBarItem(
                        selected = currentRoute == Routes.PROFILE,
                        onClick = { navController.navigate(Routes.PROFILE) { launchSingleTop = true } },
                        icon = { Icon(Icons.Default.Person, contentDescription = "我的") },
                        label = { Text("我的") },
                    )
                }
            }
        },
    ) { padding ->
        NavHost(
            navController = navController,
            startDestination = startDestination,
            modifier = Modifier.padding(padding),
        ) {
            composable(Routes.LOGIN) { LoginScreen(navController) }
            composable(Routes.REGISTER) { RegisterScreen(navController) }
            composable(Routes.DISCOVER) { DiscoverScreen(navController) }
            composable(Routes.FOLLOWING) { FollowingScreen(navController) }
            composable(Routes.PROFILE) { ProfileScreen(navController) }
            composable(Routes.PUBLISH) { PublishScreen(navController) }
            composable(
                route = Routes.DETAIL,
                arguments = listOf(navArgument(Routes.ARG_NOTE_ID) { type = NavType.LongType }),
            ) { entry ->
                DetailScreen(
                    noteId = entry.arguments?.getLong(Routes.ARG_NOTE_ID) ?: 0L,
                    navController = navController,
                )
            }
        }
    }
}
