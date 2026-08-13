package com.xbs.app.ui.detail

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Favorite
import androidx.compose.material.icons.filled.FavoriteBorder
import androidx.compose.material.icons.filled.Star
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.derivedStateOf
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavHostController
import coil.compose.AsyncImage

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DetailScreen(
    noteId: Long,
    navController: NavHostController,
    vm: DetailViewModel = hiltViewModel(),
) {
    val state by vm.uiState.collectAsStateWithLifecycle()
    val snackbar = remember { SnackbarHostState() }
    var commentInput by remember { mutableStateOf("") }

    LaunchedEffect(noteId) { vm.load() }
    LaunchedEffect(Unit) { vm.toasts.collect { snackbar.showSnackbar(it) } }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("笔记") },
                navigationIcon = {
                    IconButton(onClick = { navController.popBackStack() }) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "返回")
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbar) },
        bottomBar = {
            Row(
                Modifier.fillMaxWidth().padding(8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                OutlinedTextField(
                    value = commentInput,
                    onValueChange = { commentInput = it },
                    placeholder = { Text("说点什么...") },
                    modifier = Modifier.weight(1f),
                    singleLine = true,
                )
                Spacer(Modifier.width(8.dp))
                Button(onClick = { vm.sendComment(commentInput); commentInput = "" }) { Text("发送") }
            }
        },
    ) { padding ->
        when {
            state.isLoading && state.note == null ->
                Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            state.error != null && state.note == null ->
                Box(Modifier.fillMaxSize().padding(padding), contentAlignment = Alignment.Center) {
                    Text(state.error ?: "加载失败", color = MaterialTheme.colorScheme.error)
                }
            else -> {
                val note = state.note ?: return@Scaffold
                val listState = rememberLazyListState()
                val shouldLoadMoreComments by remember {
                    derivedStateOf {
                        val info = listState.layoutInfo
                        val lastVisible = info.visibleItemsInfo.lastOrNull()?.index ?: -1
                        info.totalItemsCount > 0 && lastVisible >= info.totalItemsCount - 3
                    }
                }
                LaunchedEffect(shouldLoadMoreComments) { if (shouldLoadMoreComments) vm.loadMoreComments() }

                LazyColumn(state = listState, modifier = Modifier.fillMaxSize().padding(padding)) {
                    item {
                        // 图片轮播
                        val pagerState = rememberPagerState(pageCount = { note.images.size.coerceAtLeast(1) })
                        HorizontalPager(state = pagerState, modifier = Modifier.fillMaxWidth().height(360.dp)) { page ->
                            AsyncImage(
                                model = note.images.getOrNull(page) ?: note.coverUrl,
                                contentDescription = null,
                                contentScale = ContentScale.Crop,
                                modifier = Modifier.fillMaxSize(),
                            )
                        }
                    }
                    item {
                        Column(Modifier.padding(16.dp)) {
                            Text(note.title, style = MaterialTheme.typography.titleLarge)
                            if (note.content.isNotBlank()) {
                                Spacer(Modifier.height(8.dp))
                                Text(note.content, style = MaterialTheme.typography.bodyMedium)
                            }
                            Spacer(Modifier.height(12.dp))
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text("作者 #${note.userId}", style = MaterialTheme.typography.bodySmall)
                                Spacer(Modifier.width(12.dp))
                                Button(onClick = vm::toggleFollow) {
                                    Text(if (state.followed) "已关注" else "关注")
                                }
                                Spacer(Modifier.weight(1f))
                                IconButton(onClick = vm::toggleLike) {
                                    Icon(
                                        if (state.liked) Icons.Default.Favorite else Icons.Default.FavoriteBorder,
                                        contentDescription = "点赞",
                                        tint = if (state.liked) MaterialTheme.colorScheme.primary else Color.Gray,
                                    )
                                }
                                Text("${state.likeCount}")
                                Spacer(Modifier.width(12.dp))
                                IconButton(onClick = vm::toggleCollect) {
                                    Icon(
                                        Icons.Default.Star,
                                        contentDescription = "收藏",
                                        tint = if (state.collected) MaterialTheme.colorScheme.primary else Color.Gray,
                                    )
                                }
                                Text("${state.collectCount}")
                            }
                            Spacer(Modifier.height(12.dp))
                            HorizontalDivider()
                            Spacer(Modifier.height(8.dp))
                            Text("评论 ${state.commentCount}", style = MaterialTheme.typography.titleSmall)
                        }
                    }
                    items(state.comments, key = { it.id }) { c ->
                        Column(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 6.dp)) {
                            Text("用户 #${c.userId}", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                            Text(c.content, style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                    item {
                        if (state.isLoadingComments) {
                            Box(Modifier.fillMaxWidth().padding(16.dp), contentAlignment = Alignment.Center) {
                                CircularProgressIndicator()
                            }
                        }
                    }
                }
            }
        }
    }
}
