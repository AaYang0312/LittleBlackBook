package com.xbs.app.ui.discover

import retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.xbs.app.MainDispatcherRule
import com.xbs.app.awaitUntil
import com.xbs.app.data.api.ErrorEnvelopeInterceptor
import com.xbs.app.data.api.NoteApi
import com.xbs.app.data.repository.NoteRepository
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.Json
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.mockwebserver.MockResponse
import okhttp3.mockwebserver.MockWebServer
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import retrofit2.Retrofit

class DiscoverViewModelTest {
    @get:Rule val mainDispatcherRule = MainDispatcherRule()

    private val server = MockWebServer()
    private lateinit var retrofit: Retrofit
    private lateinit var vm: DiscoverViewModel

    private fun noteJson(id: Long) = """
        {"id":$id,"user_id":1,"title":"n$id","content":"","cover_url":"u","images":["u"],
         "like_count":0,"collect_count":0,"comment_count":0,"created_at":"2026-08-12T10:00:00Z"}
    """.trimIndent()

    private fun pageJson(ids: List<Long>, nextCursor: Long, hasMore: Boolean) =
        """{"code":0,"message":"success","data":{"list":[${ids.joinToString(",") { noteJson(it) }}],"next_cursor":$nextCursor,"has_more":$hasMore}}"""

    @Before
    fun setUp() {
        server.start()
        val json = Json { ignoreUnknownKeys = true }
        retrofit = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient.Builder().addInterceptor(ErrorEnvelopeInterceptor()).build())
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
    }

    /** 注意：必须在 enqueue 完响应后再创建 VM——init 块会自动 refresh，先建 VM 会让首个请求无响应可用。 */
    private fun createVm() {
        vm = DiscoverViewModel(NoteRepository(retrofit.create(NoteApi::class.java)))
    }

    @After fun tearDown() = server.shutdown()

    @Test
    fun `refresh loads first page then loadMore appends until hasMore false`() = runTest {
        server.enqueue(MockResponse().setBody(pageJson(listOf(1, 2), nextCursor = 2, hasMore = true)))
        server.enqueue(MockResponse().setBody(pageJson(listOf(3), nextCursor = 3, hasMore = false)))
        createVm()  // init 自动 refresh，消费第一个响应
        awaitUntil { vm.uiState.value.items.size == 2 }
        assertEquals(listOf(1L, 2L), vm.uiState.value.items.map { it.id })

        vm.loadMore()
        awaitUntil { vm.uiState.value.items.size == 3 }
        assertEquals(listOf(1L, 2L, 3L), vm.uiState.value.items.map { it.id })
        assertFalse(vm.uiState.value.hasMore)

        vm.loadMore()  // hasMore=false，不应再发请求
        assertEquals(2, server.requestCount)
    }

    @Test
    fun `refresh replaces list`() = runTest {
        server.enqueue(MockResponse().setBody(pageJson(listOf(1), nextCursor = 1, hasMore = true)))
        server.enqueue(MockResponse().setBody(pageJson(listOf(9), nextCursor = 9, hasMore = false)))
        createVm()
        awaitUntil { vm.uiState.value.items.size == 1 }
        vm.refresh()
        awaitUntil { vm.uiState.value.items.firstOrNull()?.id == 9L }
        assertEquals(listOf(9L), vm.uiState.value.items.map { it.id })
    }

    @Test
    fun `loadMore failure sets loadMoreError and keeps items`() = runTest {
        server.enqueue(MockResponse().setBody(pageJson(listOf(1), nextCursor = 1, hasMore = true)))
        server.enqueue(MockResponse().setResponseCode(500).setBody("""{"code":-1,"message":"boom"}"""))
        createVm()
        awaitUntil { vm.uiState.value.items.size == 1 }
        vm.loadMore()
        awaitUntil { vm.uiState.value.loadMoreError }
        assertEquals(listOf(1L), vm.uiState.value.items.map { it.id })
    }
}
