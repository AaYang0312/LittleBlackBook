package com.xbs.app.ui.discover

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.staggeredgrid.LazyVerticalStaggeredGrid
import androidx.compose.foundation.lazy.staggeredgrid.StaggeredGridCells
import androidx.compose.foundation.lazy.staggeredgrid.items
import androidx.compose.foundation.lazy.staggeredgrid.rememberLazyStaggeredGridState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.pulltorefresh.PullToRefreshBox
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavHostController
import com.xbs.app.navigation.Routes
import com.xbs.app.ui.common.NoteCard

private val RATIOS = listOf(0.75f, 1.0f, 1.33f)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DiscoverScreen(
    navController: NavHostController,
    vm: DiscoverViewModel = hiltViewModel(),
) {
    val state by vm.uiState.collectAsStateWithLifecycle()
    val gridState = rememberLazyStaggeredGridState()

    // 发布页回来后自动刷新
    val published = navController.currentBackStackEntry
        ?.savedStateHandle?.getStateFlow("published", false)
        ?.collectAsStateWithLifecycle()
    LaunchedEffect(published?.value) {
        if (published?.value == true) {
            vm.refresh()
            navController.currentBackStackEntry?.savedStateHandle?.set("published", false)
        }
    }

    val shouldLoadMore by remember {
        derivedStateOf {
            val info = gridState.layoutInfo
            val lastVisible = info.visibleItemsInfo.lastOrNull()?.index ?: -1
            info.totalItemsCount > 0 && lastVisible >= info.totalItemsCount - 4
        }
    }
    LaunchedEffect(shouldLoadMore) { if (shouldLoadMore) vm.loadMore() }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("发现") },
                actions = {
                    IconButton(onClick = { navController.navigate(Routes.PUBLISH) }) {
                        Icon(Icons.Default.Add, contentDescription = "发布")
                    }
                },
            )
        },
    ) { padding ->
        PullToRefreshBox(
            isRefreshing = state.isRefreshing,
            onRefresh = vm::refresh,
            modifier = Modifier.padding(padding),
        ) {
            LazyVerticalStaggeredGrid(
                columns = StaggeredGridCells.Fixed(2),
                state = gridState,
                modifier = Modifier.fillMaxSize(),
            ) {
                items(state.items, key = { it.id }) { note ->
                    NoteCard(
                        note = note,
                        aspectRatio = RATIOS[(note.id % 3).toInt()],
                        onClick = { navController.navigate(Routes.detail(note.id)) },
                    )
                }
                item {
                    Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) {
                        when {
                            state.isLoadingMore -> CircularProgressIndicator()
                            state.loadMoreError -> TextButton(onClick = vm::loadMore) { Text("加载失败，点击重试") }
                            !state.hasMore && state.items.isNotEmpty() ->
                                Text("没有更多了", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }
        }
    }
}
