if exists('g:loaded_dbee')
    finish
endif
let g:loaded_dbee = 1

" not every error the go-client runs into is logged. Some are printed
" to stderr.  This routine captures that and prints it to messages;
" makes it easier to debug issues with code using go-client.
" You can see all messages using :messages. See :help messages for more 
" information about messages.
function! s:panic(ch, data, ...) abort
    echom a:data
endfunction

" this launches the go program we have compiled. For the above s:panic
" to work you need to have the on_error key. The 'rpc' key is to let
" nvim know to communicate with the process via msgpack-rpc over stdout.
" See :h msgpack-rpc and :h jobstart() for more information.
function! s:Start_dbee(host) abort
    return jobstart(['dbee'], {
        \ 'rpc': v:true,
        \ 'detach': v:true, 
        \ 'on_stderr': function('s:panic')
        \ })
endfunction

" these remote#host routines are in nvim/runtime/autoload/remote/host.vim in neovim. Where exactly
" this path is rooted is dependent on your nvim. e.g. in my setup the complete path is 
" /usr/share/nvim/runtime/autoload/remote/host.vim.
" You may want to read :help remote_plugin for more information.
call remote#host#Register('nvim_dbee', 'x', function('s:Start_dbee'))

" do not edit manually.
" the name for the 'host' you choose when running -manifest must match the
" name you use in the above call to remote#host#Register. e.g. in this case
" the 'host' name is 'nvim_go_client_example'.
call remote#host#RegisterPlugin('nvim_dbee', '0', [
\ {'type': 'function', 'name': 'DbeeCallCancel', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeCallDisplayResult', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeCallStoreResult', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeConnectionExecute', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeConnectionGetCalls', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeConnectionGetParams', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeConnectionGetStructure', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeConnectionListDatabases', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeConnectionSelectDatabase', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeCreateConnection', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeGetConnections', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeGetCurrentConnection', 'sync': 1, 'opts': {}},
\ {'type': 'function', 'name': 'DbeeSetCurrentConnection', 'sync': 1, 'opts': {}},
\ ])
