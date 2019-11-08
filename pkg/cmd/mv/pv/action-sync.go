package pv

import (
	"fmt"
	"path/filepath"

	"go.zoe.im/kops/pkg/utils"

	"go.zoe.im/x/sh"
	"github.com/fatih/color"
)

func HandlerSync(act *ActionConfig) error {
	// start sync service to sync data
	// **make sure pv not be deleted

	var (
		err         error
		data map[string]string
	)

	fmt.Printf("[%d] 同步数据\n", act.CurrentStep)

	// handle all pvcs
	for idx := range act.Items {
		var item = act.Items[idx]
		if !item.FirstSynced {
			fmt.Printf("    同步第 %d 个PV %s 的数据\n", idx, item.OldPv.Name)
			// first time to sync data
			// setup directory and try to create directory
			if item.SourcePath == "" {
				// FIXME: if source path don't have any data, rsync will be failed.
				item.SourcePath = item.OldPv.Spec.Local.Path + "/*"
			}

			if item.TargetPath == "" {
				_parent := utils.ParentPath(item.OldPv.Spec.Local.Path)
				// always use a new path name
				if act.m.Config.Directory != "" {
					_parent = act.m.Config.Directory
				}
				_dir := act.m.Config.Prefix + "-" + item.OldPv.ObjectMeta.ResourceVersion
				item.TargetPath = filepath.Join(_parent, _dir)
			}

			if !utils.Exits(item.TargetPath) {
				fmt.Printf("    目标目录 %s 不存在, 自动创建 ", item.TargetPath)
				err = sh.Run("mkdir -p " + item.TargetPath)
				if err != nil {
					color.Red("失败")
					fmt.Printf("    error: %s\n", err)
					return err
				}
				color.Green("成功")
			}

			data = map[string]string{
				"args":        act.m.Config.RsyncArgs,
				"source_host": act.SrcHost,
				"source_path": item.SourcePath,
				"target_path": item.TargetPath,
			}

			if len(act.m.Config.Username) > 0 {
				data["username"] = act.m.Config.Username
			}
		
			item.rsynccmd = genRsyncCmd(act.m.Config.DaemonRsync, data)
		} else {
			fmt.Printf("    再同步第 %d 个PV的数据\n", idx)
		}

		// sync data run commandrunsync:
		fmt.Printf("    ")
		err = sh.Run(item.rsynccmd)
		fmt.Printf("    启动rsync同步数据 ")
		if err != nil {
			color.Red("失败")
			return err
		}
		color.Green("成功")

		if item.FirstSynced {
			item.Synced = true
		} else {
			item.FirstSynced = true
		}
	}
	
	return nil
	// if !created {
	// 	fmt.Printf("[%d] 再次同步第 %d 个PV数据\n", step, index+1)
	// 	goto runsync
	// } else {
	// 	fmt.Printf("[%d] 同步第 %d 个PV数据\n", step, index+1)
	// }

	// _path = pvp.pv.Spec.Local.Path
	// _parent = utils.ParentPath(_path)


	// _sourcepath = _path
	// _targetpath = _parent

	// // always use a new path name
	// _sourcepath = _path + "/*"
	// _targetpath = _parent + "/" + PVNameSuffix +"-" + pvp.pv.ObjectMeta.ResourceVersion


	// 2. create parent directory of target pv
	// if !utils.Exits(_parent) {
	// 	fmt.Printf("    上一级目录 %s 不存在 ", _parent)
	// 	if !m.Config.AutoCreate {
	// 		color.Red("失败")
	// 		return ErrCancel
	// 	}
	// 	fmt.Printf("    自动创建父级路径 %s ", _parent)
	// 	err = sh.Run("mkdir -p " + _parent)
	// 	if err != nil {
	// 		color.Red("失败")
	// 		fmt.Printf("    error: %s\n", err)
	// 		return err
	// 	}
	// 	color.Green("成功")
	// }

	// if _sourcepath[len(_sourcepath)-1] == '*' {
	// 	// special directory we need to make sure target exits
	// 	if !utils.Exits(_targetpath) {
	// 		fmt.Printf("    目标目录 %s 不存在, 自动创建 ", _targetpath)
	// 		err = sh.Run("mkdir " + _targetpath)
	// 		if err != nil {
	// 			color.Red("失败")
	// 			fmt.Printf("    error: %s\n", err)
	// 			return err
	// 		}
	// 		color.Green("成功")
	// 	}
	// }

	// 3. sync data use rsync
	// data = map[string]string{
	// 	"args":        m.Config.RsyncArgs,
	// 	"source_host": act.srcHost,
	// 	"source_path": _sourcepath,
	// 	"target_path": _targetpath,
	// }
	// if len(m.Config.Username) > 0 {
	// 	data["username"] = m.Config.Username
	// }

	// pvp.rsynccmd = genRsyncCmd(m.Config.DaemonRsync, data)

	// if m.Config.DryRun {
	// 	fmt.Println("    运行命令", pvp.rsynccmd)
	// 	return nil
	// }

	// 4. set _targetpath to pv
	// pvp._targetpath = _targetpath

// runsync:
// 	fmt.Printf("    ")
// 	err = sh.Run(pvp.rsynccmd)
// 	fmt.Printf("    启动rsync同步数据 ")
// 	if err != nil {
// 		color.Red("失败")
// 		if _count < 2 {
// 			color.Yellow("    再试一次")
// 			goto runsync
// 		}
// 		return err
// 	}
// 	color.Green("成功")

	// return nil
}

// imporant!!!
func genRsyncCmd(daemon bool, data map[string]string) string {
	var cmd = "rsync"
	cmd += " " + data["args"]
	if username, ok := data["username"]; ok {
		cmd += " " + username + "@" + data["source_host"]
	} else {
		cmd += " " + data["source_host"]
	}
	cmd += ":"
	if daemon {
		cmd += ":"
	}
	cmd += data["source_path"]
	cmd += " " + data["target_path"]
	return cmd
}