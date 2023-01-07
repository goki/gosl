// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/emer/emergent/timer"
	"github.com/goki/ki/ints"
	"github.com/goki/vgpu/vgpu"
)

// note: standard one to use is plain "gosl" which should be go install'd

//go:generate ../../gosl -exclude=Update,UpdateParams,Defaults -keep /Users/oreilly/go/src/github.com/goki/mat32/fastexp.go minmax chans/chans.go chans kinase time.go neuron.go act.go learn.go layer.go

func init() {
	// must lock main thread for gpu!  this also means that vulkan must be used
	// for gogi/oswin eventually if we want gui and compute
	runtime.LockOSThread()
}

func main() {
	if vgpu.Init() != nil {
		return
	}

	gp := vgpu.NewComputeGPU()
	// vgpu.Debug = true
	gp.Config("axon")

	// gp.PropsString(true) // print

	// n := 10 // 100,000 = 2.38 CPU, 0.005939 GPU
	n := 100000 // 100,000 = 2.38 CPU, 0.005939 GPU
	maxCycles := 200

	lay := &Layer{}
	lay.Defaults()

	time := NewTime()
	time.Defaults()

	neur1 := make([]Neuron, n)
	for i := range neur1 {
		d := &neur1[i]
		lay.Act.InitActs(d)
		d.GeBase = 0.4
	}
	neur2 := make([]Neuron, n)
	for i := range neur2 {
		d := &neur2[i]
		lay.Act.InitActs(d)
		d.GeBase = 0.4
	}

	cpuTmr := timer.Time{}
	if true {
		// if false {
		cpuTmr.Start()

		for cy := 0; cy < maxCycles; cy++ {
			for i := range neur1 {
				d := &neur1[i]
				// d.Vm = lay.Act.Decay.Glong
				lay.CycleNeuron(i, d, time)
			}
			time.CycleInc()
		}

		cpuTmr.Stop()
	}

	sy := gp.NewComputeSystem("axon")
	pl := sy.NewPipeline("axon")
	pl.AddShaderFile("axon", vgpu.ComputeShader, "shaders/axon.spv")

	vars := sy.Vars()
	setp := vars.AddSet()
	setd := vars.AddSet()

	layv := setp.AddStruct("Layer", int(unsafe.Sizeof(Layer{})), 1, vgpu.Uniform, vgpu.ComputeShader)
	timev := setd.AddStruct("Time", int(unsafe.Sizeof(Time{})), 1, vgpu.Storage, vgpu.ComputeShader)
	neurv := setd.AddStruct("Neurons", int(unsafe.Sizeof(Neuron{})), n, vgpu.Storage, vgpu.ComputeShader)

	setp.ConfigVals(1) // one val per var
	setd.ConfigVals(1) // one val per var
	sy.Config()        // configures vars, allocates vals, configs pipelines..

	gpuFullTmr := timer.Time{}
	gpuFullTmr.Start()

	// this copy is pretty fast -- most of time is below
	lvl, _ := layv.Vals.ValByIdxTry(0)
	lvl.CopyFromBytes(unsafe.Pointer(lay))
	tvl, _ := timev.Vals.ValByIdxTry(0)
	tvl.CopyFromBytes(unsafe.Pointer(time))
	dvl, _ := neurv.Vals.ValByIdxTry(0)
	dvl.CopyFromBytes(unsafe.Pointer(&neur2[0]))

	// gpuFullTmr := timer.Time{}
	// gpuFullTmr.Start()

	sy.Mem.SyncToGPU()

	vars.BindDynValIdx(0, "Layer", 0)
	vars.BindDynValIdx(1, "Time", 0)
	vars.BindDynValIdx(1, "Neurons", 0)

	sy.CmdResetBindVars(sy.CmdPool.Buff, 0)

	// gpuFullTmr := timer.Time{}
	// gpuFullTmr.Start()

	gpuTmr := timer.Time{}
	gpuTmr.Start()

	pl.RunComputeWait(sy.CmdPool.Buff, n, 1, 1)
	for cy := 1; cy < maxCycles; cy++ {
		sy.CmdSubmitWait()
	}

	gpuTmr.Stop()

	sy.Mem.SyncValIdxFmGPU(1, "Neurons", 0) // this is about same as SyncToGPU
	dvl.CopyToBytes(unsafe.Pointer(&neur2[0]))

	gpuFullTmr.Stop()

	mx := ints.MinInt(n, 5)
	for i := 0; i < mx; i++ {
		d1 := &neur1[i]
		d2 := &neur2[i]
		fmt.Printf("%d\tGe1: %g\tGe2: %g\tV1: %g\tV2: %g\n", i, d1.Ge, d2.Ge, d1.Vm, d2.Vm)
	}
	fmt.Printf("\n")

	fmt.Printf("N: %d\t CPU: %6.4g\t GPU: %6.4g\t Full: %6.4g\n", n, cpuTmr.TotalSecs(), gpuTmr.TotalSecs(), gpuFullTmr.TotalSecs())

	sy.Destroy()
	gp.Destroy()
	vgpu.Terminate()
}
