package com.xbs.app.ui.following

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FollowingScreen(
    navController: NavHostController,
    vm: FollowingViewModel = hiltViewModel(),
) {
    val state by vm.uiState.collectAsStateWithLifecycle()
    val listState = rememberLazyListState()

    val shouldLoadMore by remember {
        derivedStateOf {
            val info = listState.layoutInfo
            val lastVisible = info.visibleItemsInfo.lastOrNull()?.index ?: -1
            info.totalItemsCount > 0 && lastVisible >= info.totalItemsCount - 3
        }
    }
    LaunchedEffect(shouldLoadMore) { if (shouldLoadMore) vm.loadMore() }

    PullToRefreshBox(isRefreshing = state.isRefreshing, onRefresh = vm::refresh) {
        if (state.items.isEmpty() && !state.isRefreshing) {
            Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                Text("还没有关注内容，去发现页逛逛吧", style = MaterialTheme.typography.bodyMedium)
            }
        }
        LazyColumn(
            state = listState,
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(vertical = 8.dp),
        ) {
            items(state.items, key = { it.id }) { note ->
                NoteCard(
                    note = note,
                    aspectRatio = 1.33f,
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
