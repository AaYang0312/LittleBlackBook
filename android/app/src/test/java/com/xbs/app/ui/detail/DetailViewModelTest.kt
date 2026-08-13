package com.xbs.app.ui.detail

import retrofit2.converter.kotlinx.serialization.asConverterFactory
import com.xbs.app.MainDispatcherRule
import com.xbs.app.awaitUntil
import com.xbs.app.data.api.ErrorEnvelopeInterceptor
import com.xbs.app.data.api.InteractionApi
import com.xbs.app.data.api.NoteApi
import com.xbs.app.data.repository.InteractionRepository
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
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Rule
import org.junit.Test
import retrofit2.Retrofit

class DetailViewModelTest {
    @get:Rule val mainDispatcherRule = MainDispatcherRule()

    private val server = MockWebServer()
    private lateinit var vm: DetailViewModel

    private val noteBody = """
        {"code":0,"message":"success","data":{"id":7,"user_id":2,"title":"t","content":"c",
         "cover_url":"u","images":["u"],"like_count":10,"collect_count":5,"comment_count":0,
         "created_at":"2026-08-12T10:00:00Z"}}
    """.trimIndent()

    @Before
    fun setUp() {
        server.start()
        val json = Json { ignoreUnknownKeys = true }
        val retrofit = Retrofit.Builder()
            .baseUrl(server.url("/"))
            .client(OkHttpClient.Builder().addInterceptor(ErrorEnvelopeInterceptor()).build())
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()
        vm = DetailViewModel(
            noteId = 7,
            noteRepo = NoteRepository(retrofit.create(NoteApi::class.java)),
            interactionRepo = InteractionRepository(retrofit.create(InteractionApi::class.java)),
        )
    }

    @After fun tearDown() = server.shutdown()

    @Test
    fun `toggleLike optimistic increment then confirm`() = runTest {
        server.enqueue(MockResponse().setBody(noteBody))            // detail
        server.enqueue(MockResponse().setBody("""{"code":0,"message":"success","data":[]}""")) // comments
        server.enqueue(MockResponse().setBody("""{"code":0,"message":"success"}"""))          // like
        vm.load()
        awaitUntil { vm.uiState.value.note != null }
        assertEquals(10L, vm.uiState.value.likeCount)

        vm.toggleLike()
        // 乐观更新：状态翻转在协程发起前同步完成，此断言是确定性的
        assertTrue(vm.uiState.value.liked)
        awaitUntil { vm.uiState.value.likeCount == 11L }
    }

    @Test
    fun `toggleLike rolls back on failure`() = runTest {
        server.enqueue(MockResponse().setBody(noteBody))
        server.enqueue(MockResponse().setBody("""{"code":0,"message":"success","data":[]}"""))
        server.enqueue(MockResponse().setResponseCode(500).setBody("""{"code":-1,"message":"boom"}"""))
        vm.load()
        awaitUntil { vm.uiState.value.note != null }

        vm.toggleLike()
        awaitUntil { !vm.uiState.value.liked }          // 等待回滚完成
        assertEquals(10L, vm.uiState.value.likeCount)
        assertFalse(vm.uiState.value.liked)
    }
}
