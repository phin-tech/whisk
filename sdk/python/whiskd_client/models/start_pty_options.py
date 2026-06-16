from __future__ import annotations

from collections.abc import Mapping
from typing import TYPE_CHECKING, Any, TypeVar, cast

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

if TYPE_CHECKING:
    from ..models.start_pty_agent_bridge_options import StartPTYAgentBridgeOptions
    from ..models.start_pty_options_env import StartPTYOptionsEnv


T = TypeVar("T", bound="StartPTYOptions")


@_attrs_define
class StartPTYOptions:
    """
    Attributes:
        cols (int):
        rows (int):
        agent_bridge (None | StartPTYAgentBridgeOptions | Unset):
        args (list[str] | Unset):
        command (str | Unset):
        env (StartPTYOptionsEnv | Unset):
        exec_ (bool | Unset):
    """

    cols: int
    rows: int
    agent_bridge: None | StartPTYAgentBridgeOptions | Unset = UNSET
    args: list[str] | Unset = UNSET
    command: str | Unset = UNSET
    env: StartPTYOptionsEnv | Unset = UNSET
    exec_: bool | Unset = UNSET
    additional_properties: dict[str, Any] = _attrs_field(init=False, factory=dict)

    def to_dict(self) -> dict[str, Any]:
        from ..models.start_pty_agent_bridge_options import StartPTYAgentBridgeOptions

        cols = self.cols

        rows = self.rows

        agent_bridge: dict[str, Any] | None | Unset
        if isinstance(self.agent_bridge, Unset):
            agent_bridge = UNSET
        elif isinstance(self.agent_bridge, StartPTYAgentBridgeOptions):
            agent_bridge = self.agent_bridge.to_dict()
        else:
            agent_bridge = self.agent_bridge

        args: list[str] | Unset = UNSET
        if not isinstance(self.args, Unset):
            args = self.args

        command = self.command

        env: dict[str, Any] | Unset = UNSET
        if not isinstance(self.env, Unset):
            env = self.env.to_dict()

        exec_ = self.exec_

        field_dict: dict[str, Any] = {}
        field_dict.update(self.additional_properties)
        field_dict.update(
            {
                "cols": cols,
                "rows": rows,
            }
        )
        if agent_bridge is not UNSET:
            field_dict["agentBridge"] = agent_bridge
        if args is not UNSET:
            field_dict["args"] = args
        if command is not UNSET:
            field_dict["command"] = command
        if env is not UNSET:
            field_dict["env"] = env
        if exec_ is not UNSET:
            field_dict["exec"] = exec_

        return field_dict

    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.start_pty_agent_bridge_options import StartPTYAgentBridgeOptions
        from ..models.start_pty_options_env import StartPTYOptionsEnv

        d = dict(src_dict)
        cols = d.pop("cols")

        rows = d.pop("rows")

        def _parse_agent_bridge(
            data: object,
        ) -> None | StartPTYAgentBridgeOptions | Unset:
            if data is None:
                return data
            if isinstance(data, Unset):
                return data
            try:
                if not isinstance(data, dict):
                    raise TypeError()
                agent_bridge_type_1 = StartPTYAgentBridgeOptions.from_dict(data)

                return agent_bridge_type_1
            except (TypeError, ValueError, AttributeError, KeyError):
                pass
            return cast(None | StartPTYAgentBridgeOptions | Unset, data)

        agent_bridge = _parse_agent_bridge(d.pop("agentBridge", UNSET))

        args = cast(list[str], d.pop("args", UNSET))

        command = d.pop("command", UNSET)

        _env = d.pop("env", UNSET)
        env: StartPTYOptionsEnv | Unset
        if isinstance(_env, Unset):
            env = UNSET
        else:
            env = StartPTYOptionsEnv.from_dict(_env)

        exec_ = d.pop("exec", UNSET)

        start_pty_options = cls(
            cols=cols,
            rows=rows,
            agent_bridge=agent_bridge,
            args=args,
            command=command,
            env=env,
            exec_=exec_,
        )

        start_pty_options.additional_properties = d
        return start_pty_options

    @property
    def additional_keys(self) -> list[str]:
        return list(self.additional_properties.keys())

    def __getitem__(self, key: str) -> Any:
        return self.additional_properties[key]

    def __setitem__(self, key: str, value: Any) -> None:
        self.additional_properties[key] = value

    def __delitem__(self, key: str) -> None:
        del self.additional_properties[key]

    def __contains__(self, key: str) -> bool:
        return key in self.additional_properties
