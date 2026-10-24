# build_bitstream.tcl - Fluxo em lote (Batch Mode) para Vivado
set PART      "xc7a35tcpg236-1"
set TOP_MOD   "fpga_test_top"
set OUTPUT_DIR "build"

file mkdir $OUTPUT_DIR

# 1. Leitura dos Fontes Verilog
read_verilog [ glob rtl/*.v ]

# 2. Leitura do Arquivo de Restrições Físicas (XDC)
read_xdc constr/basys3_6502t.xdc

# 3. Síntese Lógica
synth_design -top $TOP_MOD -part $PART -flatten_hierarchy rebuilt
write_checkpoint -force $OUTPUT_DIR/post_synth.dcp
report_utilization -file $OUTPUT_DIR/post_synth_util.rpt

# 4. Otimização e Posicionamento (Place)
opt_design
place_design
phys_opt_design
write_checkpoint -force $OUTPUT_DIR/post_place.dcp

# 5. Roteamento (Route)
route_design
write_checkpoint -force $OUTPUT_DIR/post_route.dcp
report_timing_summary -file $OUTPUT_DIR/post_route_timing.rpt
report_drc -file $OUTPUT_DIR/post_route_drc.rpt

# 6. Geração do Bitstream
write_bitstream -force $OUTPUT_DIR/${TOP_MOD}.bit
puts "=== BITSTREAM GERADO COM SUCESSO: $OUTPUT_DIR/${TOP_MOD}.bit ==="
exit