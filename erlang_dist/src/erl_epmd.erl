%%%-------------------------------------------------------------------
%%% @author sdhillon
%%% @copyright (C) 2015, <COMPANY>
%%% @doc
%%%
%%% @end
%%% Created : 03. Aug 2015 9:19 PM
%%%-------------------------------------------------------------------
-module(erl_epmd).
-author("sdhillon").



%% API
-export([register_node/2, port_please/2, port_please/3, mesos_mode/0, get_port/1]).
-export([wait_forever/1, wait_forever/0]).

-export([start/0, start_link/0]).

wait_forever() ->
    receive
        _ ->
            wait_forever()
    end.

start() ->
    {ok, spawn(?MODULE, wait_forever, [])}.

start_link() ->
    {ok, spawn_link(?MODULE, wait_forever, [])}.
mesos_mode() -> true.
wait_forever(Socket) ->
    receive
        Stuff ->
            erlang:display({unknown_cepmd_info, Stuff}),
            wait_forever(Socket)
    end.

register_node(Name, PortNo) ->
    erlang:display({register, Name}),
    {ok, Conn} = get_connect(),
    Line = io_lib:format("REGISTER ~s ~B~n", [Name, PortNo]),
    LineBin =  binary:list_to_bin(Line),
    gen_tcp:send(Conn, LineBin),
    Wait = do_wait_for_data(Conn),
    case Wait of
        {ok, "OK\n"} ->
            Pid = erlang:spawn(?MODULE, wait_forever, [Conn]),
            gen_tcp:controlling_process(Conn, Pid),
            {ok, 0};
        Else ->
            gen_tcp:close(Conn),
            Else
    end.

%% Lookup a node "Name" at Host
%% return {port, P, Version} | noport
%%
%% Version = 5
port_please(Node, _HostName) ->
    get_port(Node).

port_please(Node, _HostName, _Timeout) ->
    get_port(Node).
get_port(Node) ->
    {ok, Conn} = get_connect(),
    Line = io_lib:format("PORT_PLEASE ~s~n", [Node]),
    LineBin =  binary:list_to_bin(Line),
    gen_tcp:send(Conn, LineBin),
    Wait = do_wait_for_data(Conn),
    gen_tcp:close(Conn),
    case Wait of
        {ok, "NOTFOUND\n"} ->
            noport;
        {ok, PortStr} ->
            {Port, _} = string:to_integer(PortStr),
            {port, Port, 5};
        _Else ->
            noport
    end.
   % get_port(Node,HostName, Timeout).

get_port1() ->
    case application:get_env(kernel, cepmd_port) of
        undefined ->
            get_port2();
        {ok, Port} ->
            {ok, Port}
    end.

get_port2() ->
    case os:getenv("CEPMD_PORT") of
        false ->
            get_port3();
        CEPMDPortStr ->
            {PortInt, _} = string:to_integer(CEPMDPortStr),
            {ok, PortInt}
    end.
get_port3() ->
    File = filename:join([code:priv_dir(kernel), "cepmd_port"]),
    case file:consult(File) of
        {ok,[Port]} ->
            {ok, Port};
        _ ->
            false
    end.

%% TODO: Make IPv6 friendly
get_connect() ->
    case get_port1() of
        {ok, Port} ->
            %% TODO: Make IPv6 friendly
            gen_tcp:connect({127,0,0,1}, Port, [{mode, list}, {packet, line}, {active, once}]);
        _Else ->
            {error, no_cepmd_port}
    end.

do_wait_for_data(Conn) ->
    receive
        {tcp, Conn, Data} ->
            {ok, Data};
        {tcp_passive, Conn} ->
            {error, tcp_passive};
        {tcp_closed, Conn} ->
            {error, socket_closed};
        {tcp_error, Conn, Reason} ->
            {error, Reason}
    after 5000 ->
        erlang:display("Error, timed out waiting for data"),
        {error, timeout}
    end.

